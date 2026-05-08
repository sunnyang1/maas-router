"""
Online Learner Module

Provides online learning capabilities for adjusting model tier thresholds
based on quality feedback. Uses simple moving average logic to gradually
improve routing decisions.
"""

import time
import logging
from typing import Optional, Dict, Any, List, Tuple
from collections import deque

from .models import ModelTierConfig, FeedbackRequest

logger = logging.getLogger(__name__)


class FeedbackRecord:
    """A single feedback record for tracking quality."""

    def __init__(self, request_id: str, quality_score: float, tier: str, model: str, timestamp: float):
        self.request_id = request_id
        self.quality_score = quality_score
        self.tier = tier
        self.model = model
        self.timestamp = timestamp


class TierStats:
    """Statistics for a single model tier."""

    def __init__(self, tier: str, initial_threshold: float):
        self.tier = tier
        self.threshold = initial_threshold
        self.feedback_history: deque = deque(maxlen=1000)  # Recent feedback scores
        self.total_requests = 0
        self.total_feedback = 0
        self.avg_quality = 0.0
        self.pass_rate = 0.0  # Rate of requests with quality >= 0.7
        self.last_adjusted_at = 0.0

    def add_feedback(self, quality_score: float):
        """Record a new feedback score."""
        self.feedback_history.append(quality_score)
        self.total_feedback += 1
        self._recalculate()

    def _recalculate(self):
        """Recalculate statistics from feedback history."""
        if not self.feedback_history:
            return

        scores = list(self.feedback_history)
        self.avg_quality = sum(scores) / len(scores)
        self.pass_rate = sum(1 for s in scores if s >= 0.7) / len(scores)

    def should_adjust(self, min_samples: int = 20, min_interval_sec: int = 300) -> bool:
        """Check if the tier threshold should be adjusted."""
        if len(self.feedback_history) < min_samples:
            return False
        if time.time() - self.last_adjusted_at < min_interval_sec:
            return False
        return True

    def get_adjusted_threshold(self, min_pass_rate: float = 0.9) -> float:
        """
        Calculate an adjusted threshold based on quality feedback.

        If pass rate is below target, increase threshold (route more to higher tiers).
        If pass rate is well above target, decrease threshold (can use lower tiers more).
        """
        if not self.feedback_history:
            return self.threshold

        adjustment = 0.0

        # Adjust based on pass rate
        if self.pass_rate < min_pass_rate:
            # Quality is too low, increase threshold to route to higher tiers
            deficit = min_pass_rate - self.pass_rate
            adjustment = deficit * 0.1  # Gradual adjustment
        elif self.pass_rate > min_pass_rate + 0.05:
            # Quality is very good, we can be more aggressive with lower tiers
            surplus = self.pass_rate - min_pass_rate
            adjustment = -surplus * 0.05  # Smaller decrease to be conservative

        # Clamp adjustment to avoid drastic changes
        adjustment = max(-0.1, min(0.1, adjustment))

        new_threshold = self.threshold + adjustment
        # Keep threshold in reasonable range
        new_threshold = max(0.1, min(1.0, new_threshold))

        self.last_adjusted_at = time.time()
        self.threshold = new_threshold

        logger.info(
            f"Tier '{self.tier}' threshold adjusted: "
            f"{self.threshold - adjustment:.4f} -> {new_threshold:.4f} "
            f"(pass_rate={self.pass_rate:.3f}, avg_quality={self.avg_quality:.3f})"
        )

        return new_threshold


class OnlineLearner:
    """
    Online learner for adjusting model tier thresholds based on quality feedback.

    Uses simple moving average logic to gradually improve routing decisions.
    """

    def __init__(
        self,
        initial_tiers: Optional[List[ModelTierConfig]] = None,
        min_pass_rate: float = 0.9,
        auto_upgrade_threshold: float = 0.85,
        feedback_sample_rate: float = 0.05,
        stats_window_sec: int = 3600,
    ):
        self.min_pass_rate = min_pass_rate
        self.auto_upgrade_threshold = auto_upgrade_threshold
        self.feedback_sample_rate = feedback_sample_rate
        self.stats_window_sec = stats_window_sec

        # Initialize tier stats from configuration
        self.tier_stats: Dict[str, TierStats] = {}
        if initial_tiers:
            for tier_config in initial_tiers:
                self.tier_stats[tier_config.tier] = TierStats(
                    tier=tier_config.tier,
                    initial_threshold=tier_config.threshold,
                )

        # Pending feedback queue (request_id -> metadata)
        self._pending_feedback: Dict[str, Dict[str, Any]] = {}

        # Recent feedback records for analysis
        self._recent_feedback: deque = deque(maxlen=10000)

    def track_request(self, request_id: str, tier: str, model: str):
        """
        Track a request for potential feedback.

        Call this when a request is routed, so we can later correlate feedback.
        """
        self._pending_feedback[request_id] = {
            "tier": tier,
            "model": model,
            "timestamp": time.time(),
        }

        # Clean up old pending entries (older than 1 hour)
        cutoff = time.time() - 3600
        expired = [rid for rid, meta in self._pending_feedback.items() if meta["timestamp"] < cutoff]
        for rid in expired:
            del self._pending_feedback[rid]

    async def record_feedback(self, request_id: str, quality_score: float):
        """
        Record quality feedback for a previously tracked request.

        Args:
            request_id: The request ID from the original routing decision
            quality_score: Quality score from 0.0 to 1.0
        """
        # Get the original routing metadata
        metadata = self._pending_feedback.get(request_id)
        if metadata is None:
            logger.debug(f"No routing metadata found for request {request_id}, skipping feedback")
            return

        tier = metadata["tier"]
        model = metadata["model"]
        timestamp = metadata["timestamp"]

        # Record the feedback
        record = FeedbackRecord(
            request_id=request_id,
            quality_score=quality_score,
            tier=tier,
            model=model,
            timestamp=timestamp,
        )
        self._recent_feedback.append(record)

        # Update tier statistics
        if tier in self.tier_stats:
            self.tier_stats[tier].add_feedback(quality_score)
            self.tier_stats[tier].total_requests += 1

            # Check if we should adjust thresholds
            if self.tier_stats[tier].should_adjust():
                self.tier_stats[tier].get_adjusted_threshold(self.min_pass_rate)

        # Clean up the pending entry
        del self._pending_feedback[request_id]

        logger.info(
            f"Feedback recorded: request_id={request_id}, "
            f"tier={tier}, model={model}, quality={quality_score:.3f}"
        )

    async def get_adjusted_tiers(self) -> List[ModelTierConfig]:
        """
        Get the current adjusted tier configurations.

        Returns ModelTierConfig objects with potentially adjusted thresholds.
        """
        result = []
        for tier_name, stats in self.tier_stats.items():
            config = ModelTierConfig(
                tier=tier_name,
                models=[],  # Models are managed by configuration, not the learner
                threshold=stats.threshold,
                cost_per_token=0.0,  # Cost is managed by configuration
                fallback_model="",
            )
            result.append(config)
        return result

    def get_tier_stats(self) -> Dict[str, Dict[str, Any]]:
        """Get statistics for all tiers."""
        result = {}
        for tier_name, stats in self.tier_stats.items():
            result[tier_name] = {
                "threshold": stats.threshold,
                "total_requests": stats.total_requests,
                "total_feedback": stats.total_feedback,
                "avg_quality": round(stats.avg_quality, 4),
                "pass_rate": round(stats.pass_rate, 4),
                "feedback_count": len(stats.feedback_history),
            }
        return result

    def should_auto_upgrade(self, tier: str) -> bool:
        """
        Check if requests to a given tier should be auto-upgraded.

        Auto-upgrade is triggered when the pass rate drops below the threshold.
        """
        if tier not in self.tier_stats:
            return False

        stats = self.tier_stats[tier]
        if len(stats.feedback_history) < 20:
            return False

        return stats.pass_rate < self.auto_upgrade_threshold

    def get_quality_summary(self) -> Dict[str, Any]:
        """Get a summary of quality metrics across all tiers."""
        total_feedback = sum(s.total_feedback for s in self.tier_stats.values())
        total_pass = sum(
            sum(1 for s in stats.feedback_history if s >= 0.7)
            for stats in self.tier_stats.values()
        )

        overall_pass_rate = total_pass / total_feedback if total_feedback > 0 else 0.0

        return {
            "total_feedback": total_feedback,
            "overall_pass_rate": round(overall_pass_rate, 4),
            "tiers": self.get_tier_stats(),
            "auto_upgrade_candidates": [
                tier for tier in self.tier_stats
                if self.should_auto_upgrade(tier)
            ],
        }


# Global learner instance
_learner: Optional[OnlineLearner] = None


def init_learner(
    initial_tiers: Optional[List[ModelTierConfig]] = None,
    min_pass_rate: float = 0.9,
    auto_upgrade_threshold: float = 0.85,
    feedback_sample_rate: float = 0.05,
    stats_window_sec: int = 3600,
):
    """Initialize the global learner instance."""
    global _learner
    _learner = OnlineLearner(
        initial_tiers=initial_tiers,
        min_pass_rate=min_pass_rate,
        auto_upgrade_threshold=auto_upgrade_threshold,
        feedback_sample_rate=feedback_sample_rate,
        stats_window_sec=stats_window_sec,
    )
    logger.info("Online learner initialized")


def get_learner() -> OnlineLearner:
    """Get the global learner instance."""
    global _learner
    if _learner is None:
        _learner = OnlineLearner()
    return _learner
