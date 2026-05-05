"""
Async background workers for non-critical side effects.

These workers handle tasks that don't need to block the response:
- Transaction recording
- Audit logging
- Request logging
"""
