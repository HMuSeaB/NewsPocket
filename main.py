"""
NewsPocket - Main Entry Point
项目主入口，协调抓取、解析和发送流程
"""

import json
import logging
import os
import sys
from pathlib import Path

# 添加 src 到路径以便导入
sys.path.append(str(Path(__file__).parent))

from src.fetcher import RSSFetcher
from src.parser import ContentParser
from src.email_sender import EmailSender

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout)
    ]
)

logger = logging.getLogger("Main")

def load_config(config_path: str = 'config/sources.json'):
    """加载配置文件"""
    try:
        with open(config_path, 'r', encoding='utf-8') as f:
            return json.load(f)
    except Exception as e:
        logger.error(f"加载配置文件失败: {e}")
        return None

def main():
    logger.info("=== NewsPocket 开始运行 ===")

    # 1. 加载配置
    base_dir = Path(__file__).parent
    base_dir = Path(__file__).parent
    config_path = base_dir / 'config' / 'sources.json'

    config = load_config(str(config_path))
    if not config:
        sys.exit(1)

    sources = config.get('sources', [])
    source_configs = {s.get('name'): s for s in sources}
    settings = config.get('settings', {})

    max_items = settings.get('max_items_per_source', 5)
    hours_lookback = settings.get('hours_lookback', 24)
    summary_len = settings.get('summary_max_length', 200)

    logger.info(f"加载了 {len(sources)} 个源配置")

    # 2. 抓取内容
    fetcher = RSSFetcher(timeout=20, max_workers=10)
    fetch_results = fetcher.fetch_all(sources)
    fetcher.close()

    if not fetch_results:
        logger.warning("未抓取到任何内容，程序结束")
        sys.exit(0)

    # 3. 解析和清洗
    parser = ContentParser(
        max_summary_length=summary_len,
        hours_lookback=hours_lookback
    )

    all_items = parser.parse_all(fetch_results, max_items_per_source=max_items)

    if not all_items:
        logger.warning("解析后无有效内容（可能是所有内容都过期了），程序结束")
        sys.exit(0)

    # 4. 分组和统计
    news_by_category = parser.group_by_category(all_items)

    stats = {
        'total': len(all_items),
        'sources': len(set(item['source'] for item in all_items)),
        'categories': len(news_by_category)
    }

    logger.info(f"统计信息: 共 {stats['total']} 条新闻，来自 {stats['sources']} 个源，覆盖 {stats['categories']} 个分类")

    # 5. 发送邮件
    # 如果是测试模式，只生成文件不发送
    if '--test' in sys.argv:
        logger.info("测试模式：生成 output.html")
        sender = EmailSender(template_dir=str(base_dir / 'templates'))

        # 模拟环境上下文
        from datetime import datetime, timedelta, timezone
        beijing_tz = timezone(timedelta(hours=8))
        today = datetime.now(beijing_tz).strftime('%Y年%m月%d日 %A')

        context = {
            'title': f'NewsPocket 晨报 - {today} (测试)',
            'date': today,
            'news_by_category': news_by_category,
            'total_count': stats['total'],
            'source_count': stats['sources'],
            'category_count': stats['categories'],
            'source_configs': source_configs
        }

        html = sender.render_template('email_template.html', context)
        with open('output.html', 'w', encoding='utf-8') as f:
            f.write(html)
        logger.info("测试文件已生成: output.html")
    else:
        # 检查必要的环境变量
        if not os.environ.get('EMAIL_USER') or not os.environ.get('EMAIL_PASS'):
            logger.error("缺失环境变量 EMAIL_USER 或 EMAIL_PASS，无法发送邮件")
            # 在 GitHub Actions 中，这应该导致失败
            if os.environ.get('GITHUB_ACTIONS'):
                sys.exit(1)
            else:
                logger.info("本地运行请设置环境变量或使用 --test 参数")
                sys.exit(1)

        sender = EmailSender(template_dir=str(base_dir / 'templates'))
        sender.send_daily_news(news_by_category, stats, source_configs)

    logger.info("=== NewsPocket 运行完成 ===")

if __name__ == "__main__":
    main()
