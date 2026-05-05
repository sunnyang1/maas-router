"""
Seed database using ORM models.
Run: python -m app.scripts.seed
"""
import asyncio
import json

from app.core.database import async_session_factory, engine, Base
from app.core.security import hash_password, generate_api_key
from app.models.user import User
from app.models.api_key import ApiKey
from app.models.billing import Balance
from app.models.provider import Provider, Model


async def seed():
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    async with async_session_factory() as db:
        # ========== Providers ==========
        providers_data = [
            Provider(id="openai", name="OpenAI", logo_url="/icons/openai.svg", description="GPT-4o, GPT-4o Mini 等前沿模型", api_base_url="https://api.openai.com/v1", config={"api_key_env": "OPENAI_API_KEY"}),
            Provider(id="anthropic", name="Anthropic", logo_url="/icons/anthropic.svg", description="Claude 系列安全可控的 AI 模型", api_base_url="https://api.anthropic.com/v1", config={"api_key_env": "ANTHROPIC_API_KEY"}),
            Provider(id="google", name="Google AI", logo_url="/icons/google.svg", description="Gemini 系列多模态大模型", api_base_url="https://generativelanguage.googleapis.com/v1beta", config={"api_key_env": "GOOGLE_API_KEY"}),
            Provider(id="meta", name="Meta", logo_url="/icons/meta.svg", description="Llama 系列开源大模型", config={}),
            Provider(id="deepseek", name="DeepSeek", logo_url="/icons/deepseek.svg", description="高性价比国产大模型", api_base_url="https://api.deepseek.com/v1", config={"api_key_env": "DEEPSEEK_API_KEY"}),
            Provider(id="mistral", name="Mistral AI", logo_url="/icons/mistral.svg", description="欧洲领先开源模型", api_base_url="https://api.mistral.ai/v1", config={"api_key_env": "MISTRAL_API_KEY"}),
            Provider(id="alibaba", name="Alibaba Cloud", logo_url="/icons/alibaba.svg", description="通义千问 Qwen 系列模型", api_base_url="https://dashscope.aliyuncs.com/api/v1", config={"api_key_env": "DASHSCOPE_API_KEY"}),
            Provider(id="self-hosted", name="自建集群", logo_url="/icons/self.svg", description="自建 DeepSeek-V4 推理集群", api_base_url="http://inference-exec:8003/v1", config={"max_concurrent": 100, "node_count": 1}),
        ]
        for p in providers_data:
            existing = await db.get(Provider, p.id)
            if not existing:
                db.add(p)
        await db.flush()

        # ========== Models ==========
        models_data = [
            Model(id="gpt-4o", provider_id="openai", name="GPT-4o", description="最先进的多模态模型", tags=["chat","multimodal","reasoning"], context_window=128000, input_price=2.50, output_price=10.00, features=["多模态","函数调用","JSON模式"], popularity=95, is_recommended=True),
            Model(id="gpt-4o-mini", provider_id="openai", name="GPT-4o Mini", description="轻量高效的版本", tags=["chat","fast","cost-effective"], context_window=128000, input_price=0.15, output_price=0.60, features=["函数调用","JSON模式"], popularity=90),
            Model(id="claude-3.5-sonnet", provider_id="anthropic", name="Claude 3.5 Sonnet", description="最强大的 Claude 模型", tags=["chat","reasoning","code"], context_window=200000, input_price=3.00, output_price=15.00, features=["超长上下文","代码生成"], popularity=92, is_recommended=True),
            Model(id="gemini-1.5-pro", provider_id="google", name="Gemini 1.5 Pro", description="200万上下文窗口", tags=["chat","multimodal","long-context"], context_window=2000000, input_price=1.25, output_price=5.00, features=["超长上下文","多模态"], popularity=85),
            Model(id="gemini-1.5-flash", provider_id="google", name="Gemini 1.5 Flash", description="轻量快速版本", tags=["chat","fast"], context_window=1000000, input_price=0.075, output_price=0.30, features=["快速响应"], popularity=78),
            Model(id="llama-3.1-405b", provider_id="meta", name="Llama 3.1 405B", description="Meta 最大开源模型", tags=["chat","opensource"], context_window=128000, input_price=2.00, output_price=3.00, features=["开源","高性能"], popularity=75),
            Model(id="llama-3.1-70b", provider_id="meta", name="Llama 3.1 70B", description="高效版本", tags=["chat","opensource","cost-effective"], context_window=128000, input_price=0.59, output_price=0.79, features=["开源","高性价比"], popularity=78),
            Model(id="deepseek-v3", provider_id="deepseek", name="DeepSeek-V3", description="最新旗舰模型", tags=["chat","reasoning","code"], context_window=128000, input_price=0.27, output_price=1.10, features=["高性能","低价格","中文优化"], popularity=88, is_recommended=True),
            Model(id="deepseek-coder", provider_id="deepseek", name="DeepSeek-Coder", description="专业代码生成", tags=["code","chat"], context_window=64000, input_price=0.14, output_price=0.28, features=["代码生成"], popularity=82),
            Model(id="mixtral-8x22b", provider_id="mistral", name="Mixtral 8x22B", description="最大 MoE 模型", tags=["chat","opensource","moe"], context_window=65536, input_price=0.90, output_price=0.90, features=["MoE架构","多语言"], popularity=72),
            Model(id="qwen2-72b", provider_id="alibaba", name="Qwen2-72B", description="通义千问旗舰模型", tags=["chat","reasoning"], context_window=131072, input_price=0.53, output_price=1.06, features=["中文优化"], popularity=76),
            Model(id="qwen-vl-plus", provider_id="alibaba", name="Qwen-VL-Plus", description="视觉语言增强模型", tags=["multimodal","vision"], context_window=32768, input_price=1.13, output_price=2.26, features=["视觉理解"], popularity=68),
            Model(id="deepseek-v4-self", provider_id="self-hosted", name="DeepSeek-V4 (自建)", description="自建推理集群，成本节省 50%", tags=["chat","cost-effective"], context_window=128000, input_price=0.15, output_price=0.50, features=["极低成本","低延迟"], popularity=70, is_recommended=True),
        ]
        for m in models_data:
            existing = await db.get(Model, m.id)
            if not existing:
                db.add(m)
        await db.flush()

        # ========== Admin User ==========
        admin_id = "a0000000-0000-0000-0000-000000000001"
        existing_admin = await db.get(User, admin_id)
        if not existing_admin:
            admin = User(id=admin_id, email="admin@maas-router.com", password_hash=hash_password("admin123"), display_name="超级管理员", plan_id="enterprise", email_verified=True)
            db.add(admin)
            db.add(Balance(user_id=admin_id, cred_balance=100000.0))

            full_key, prefix, key_hash = generate_api_key()
            db.add(ApiKey(user_id=admin_id, name="Default Admin Key", key_prefix=prefix, key_hash=key_hash))
        await db.flush()

        # ========== Demo User ==========
        demo_id = "a0000000-0000-0000-0000-000000000002"
        existing_demo = await db.get(User, demo_id)
        if not existing_demo:
            demo = User(id=demo_id, email="demo@maas-router.com", password_hash=hash_password("demo123"), display_name="Demo 开发者", plan_id="free", email_verified=True)
            db.add(demo)
            db.add(Balance(user_id=demo_id, cred_balance=100.0))

            full_key2, prefix2, key_hash2 = generate_api_key()
            db.add(ApiKey(user_id=demo_id, name="Demo Key", key_prefix=prefix2, key_hash=key_hash2))

        await db.commit()

    print("✅ Seed data inserted successfully!")


if __name__ == "__main__":
    asyncio.run(seed())
