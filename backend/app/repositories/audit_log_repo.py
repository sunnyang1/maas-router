"""
Audit log repository.
"""
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.routing import AuditLog
from app.repositories.base import BaseRepository


class AuditLogRepository(BaseRepository[AuditLog]):
    def __init__(self, session: AsyncSession):
        super().__init__(AuditLog, session)
