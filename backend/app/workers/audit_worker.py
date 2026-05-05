"""
Background audit worker.

Records audit log entries asynchronously.
"""
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import async_session_factory
from app.models.routing import AuditLog
from app.repositories.audit_log_repo import AuditLogRepository


async def record_audit_async(
    action: str,
    user_id: str | None = None,
    resource_type: str | None = None,
    resource_id: str | None = None,
    old_value: dict | None = None,
    new_value: dict | None = None,
    ip_address: str | None = None,
    user_agent: str | None = None,
) -> AuditLog | None:
    """Record an audit log entry asynchronously."""
    async with async_session_factory() as session:
        try:
            repo = AuditLogRepository(session)
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
            await repo.create(entry)
            await session.commit()
            return entry
        except Exception:
            await session.rollback()
            return None
