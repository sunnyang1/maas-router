"""
Audit service - records system audit events.
"""
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.routing import AuditLog
from app.repositories.audit_log_repo import AuditLogRepository


class AuditService:
    """Records audit trail entries for admin actions."""

    def __init__(self, session: AsyncSession):
        self.repo = AuditLogRepository(session)

    async def log(
        self,
        action: str,
        user_id: str | None = None,
        resource_type: str | None = None,
        resource_id: str | None = None,
        old_value: dict | None = None,
        new_value: dict | None = None,
        ip_address: str | None = None,
        user_agent: str | None = None,
    ) -> AuditLog:
        """Record an audit event."""
        entry = AuditLog(
            action=action,
            user_id=user_id,
            resource_type=resource_type,
            resource_id=resource_id,
            old_value=old_value,
            new_value=new_value,
            ip_address=ip_address,
            user_agent=user_agent,
        )
        await self.repo.create(entry)
        return entry
