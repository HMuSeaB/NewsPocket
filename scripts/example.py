"""
示例爬虫脚本
模拟抓取数据，演示自定义脚本功能
"""

import json
from datetime import datetime

def fetch_data():
    """
    抓取数据函数
    必须返回列表，列表项应包含 title, link, summary, published 字段
    """
    print("正在执行示例爬虫...")

    current_time = datetime.now().strftime('%Y-%m-%d %H:%M:%S')

    # 模拟数据
    data = [
        {
            "title": "示例新闻 1: Python 脚本抓取成功",
            "link": "https://github.com/newspocket",
            "summary": "这是一条由自定义 Python 脚本生成的新闻条目，用于演示扩展功能。",
            "published": current_time
        },
        {
            "title": "示例新闻 2: 支持动态执行",
            "link": "https://python.org",
            "summary": "自定义脚本可以执行任意逻辑，例如爬取复杂网页、调用私有 API 或处理本地数据。",
            "published": current_time
        }
    ]

    return data

if __name__ == "__main__":
    # 本地测试
    print(json.dumps(fetch_data(), indent=2, ensure_ascii=False))
