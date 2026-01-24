"""
NewsPocket - Fetcher Module
并发抓取 RSS 源、JSON API 和自定义脚本，支持超时和容错处理
"""

import importlib.util
import json
import logging
import subprocess
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import List, Dict, Any, Optional
from datetime import datetime, timedelta, timezone
from pathlib import Path

import feedparser
import requests

logger = logging.getLogger(__name__)

# 默认请求头
DEFAULT_HEADERS = {
    'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
    'Accept': 'application/json, application/rss+xml, application/xml, text/xml, */*',
    'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
}


class RSSFetcher:
    """RSS 源和 JSON API 并发抓取器"""

    def __init__(self, timeout: int = 15, max_workers: int = 10):
        """
        初始化抓取器

        Args:
            timeout: 单个请求超时时间(秒)
            max_workers: 最大并发线程数
        """
        self.timeout = timeout
        self.max_workers = max_workers
        self.session = requests.Session()
        self.session.headers.update(DEFAULT_HEADERS)

    def _get_nested_value(self, data: Any, path: str) -> Any:
        """
        通过点号分隔的路径获取嵌套值
        例如: "data.list" 从 {"data": {"list": [...]}} 获取列表
        """
        if not path:
            return data

        keys = path.split('.')
        current = data
        for key in keys:
            if isinstance(current, dict):
                current = current.get(key)
            elif isinstance(current, list) and key.isdigit():
                idx = int(key)
                current = current[idx] if idx < len(current) else None
            else:
                return None
            if current is None:
                return None
        return current

    def _build_link(self, item: Dict, link_config: str, item_data: Dict) -> str:
        """
        构建条目链接
        支持直接字段名或模板格式 (如 "https://example.com/{id}")
        """
        if not link_config:
            return ""

        # 模板格式
        if '{' in link_config:
            link = link_config
            for key, value in item_data.items():
                link = link.replace(f'{{{key}}}', str(value) if value else '')
            return link

        # 直接字段名
        return str(item_data.get(link_config, ''))

    def _fetch_json_api(self, source: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        抓取 JSON API 源

        Args:
            source: 源配置字典

        Returns:
            包含源信息和条目的字典，失败返回 None
        """
        name = source.get('name', 'Unknown')
        url = source.get('url', '')
        method = source.get('method', 'GET').upper()
        json_config = source.get('json_config', {})

        # 合并 headers (源配置覆盖默认)
        headers = {**DEFAULT_HEADERS, **source.get('headers', {})}

        try:
            logger.info(f"[{name}] 开始抓取 JSON API: {url}")

            if method == 'POST':
                body = source.get('body', {})
                response = self.session.post(url, headers=headers, json=body, timeout=self.timeout)
            else:
                response = self.session.get(url, headers=headers, timeout=self.timeout)

            response.raise_for_status()
            data = response.json()

            # 提取条目列表
            items_path = json_config.get('items_path', '')
            raw_items = self._get_nested_value(data, items_path)

            if not isinstance(raw_items, list):
                logger.warning(f"[{name}] items_path '{items_path}' 未返回列表")
                return None

            # 映射字段
            title_field = json_config.get('title_field', 'title')
            link_field = json_config.get('link_field', '')
            link_template = json_config.get('link_template', '')
            time_field = json_config.get('time_field', '')
            summary_field = json_config.get('summary_field', '')

            entries = []
            for item in raw_items:
                if not isinstance(item, dict):
                    continue

                title = item.get(title_field, '')
                if not title:
                    continue

                # 构建链接
                link = ''
                if link_template:
                    link = self._build_link(item, link_template, item)
                elif link_field:
                    link = item.get(link_field, '')

                entry = {
                    'title': title,
                    'link': link,
                    'summary': item.get(summary_field, '') if summary_field else '',
                    'published': item.get(time_field, '') if time_field else '',
                    '_source_type': 'json_api',
                    '_raw': item  # 保留原始数据供调试
                }
                entries.append(entry)

            logger.info(f"[{name}] 成功获取 {len(entries)} 条内容")

            return {
                'source': source,
                'entries': entries,
                'fetched_at': datetime.now(timezone.utc)
            }

        except requests.Timeout:
            logger.error(f"[{name}] 请求超时")
        except requests.RequestException as e:
            logger.error(f"[{name}] 请求失败: {e}")
        except ValueError as e:
            logger.error(f"[{name}] JSON 解析失败: {e}")
        except Exception as e:
            logger.error(f"[{name}] 未知错误: {e}")

        return None

    def _fetch_rss(self, source: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        抓取 RSS 源 (原有逻辑)
        """
        name = source.get('name', 'Unknown')
        url = source.get('url', '')

        # 支持 RSS 源也配置自定义 headers
        headers = {**DEFAULT_HEADERS, **source.get('headers', {})}

        try:
            logger.info(f"[{name}] 开始抓取: {url}")

            response = self.session.get(url, headers=headers, timeout=self.timeout)
            response.raise_for_status()

            feed = feedparser.parse(response.content)

            if feed.bozo and not feed.entries:
                logger.warning(f"[{name}] 解析警告: {feed.bozo_exception}")
                return None

            entries_count = len(feed.entries)
            logger.info(f"[{name}] 成功获取 {entries_count} 条内容")

            return {
                'source': source,
                'feed': feed,
                'entries': feed.entries,
                'fetched_at': datetime.now(timezone.utc)
            }

        except requests.Timeout:
            logger.error(f"[{name}] 请求超时")
        except requests.RequestException as e:
            logger.error(f"[{name}] 请求失败: {e}")
        except Exception as e:
            logger.error(f"[{name}] 未知错误: {e}")

        return None

    def _fetch_script(self, source: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        执行自定义脚本抓取

        支持 Python 和 JavaScript 脚本
        """
        name = source.get('name', 'Unknown')
        script_config = source.get('script_config', {})
        script_type = script_config.get('type', 'python')
        script_path = script_config.get('path', '')
        function_name = script_config.get('function', 'fetch_data')

        if not script_path:
            logger.warning(f"[{name}] 脚本路径为空，跳过")
            return None

        # 获取绝对路径
        if not Path(script_path).is_absolute():
            script_path = str(Path(__file__).parent.parent / script_path)

        if not Path(script_path).exists():
            logger.error(f"[{name}] 脚本文件不存在: {script_path}")
            return None

        try:
            logger.info(f"[{name}] 执行 {script_type} 脚本: {script_path}")

            if script_type == 'python':
                # 动态加载 Python 模块并执行函数
                spec = importlib.util.spec_from_file_location("custom_script", script_path)
                module = importlib.util.module_from_spec(spec)
                spec.loader.exec_module(module)

                if not hasattr(module, function_name):
                    logger.error(f"[{name}] 脚本中未找到函数: {function_name}")
                    return None

                func = getattr(module, function_name)
                entries = func()

            elif script_type == 'javascript':
                # 通过 Node.js 执行 JavaScript
                js_code = f"""
                const script = require('{script_path.replace(chr(92), "/")}');
                const result = script.{function_name}();
                Promise.resolve(result).then(data => console.log(JSON.stringify(data)));
                """
                result = subprocess.run(
                    ['node', '-e', js_code],
                    capture_output=True,
                    text=True,
                    timeout=self.timeout
                )

                if result.returncode != 0:
                    logger.error(f"[{name}] JS 脚本执行失败: {result.stderr}")
                    return None

                entries = json.loads(result.stdout)

            else:
                logger.error(f"[{name}] 不支持的脚本类型: {script_type}")
                return None

            # 标记来源类型
            if isinstance(entries, list):
                for entry in entries:
                    if isinstance(entry, dict):
                        entry['_source_type'] = 'script'

            logger.info(f"[{name}] 脚本返回 {len(entries) if isinstance(entries, list) else 0} 条内容")

            return {
                'source': source,
                'entries': entries if isinstance(entries, list) else [],
                'fetched_at': datetime.now(timezone.utc)
            }

        except subprocess.TimeoutExpired:
            logger.error(f"[{name}] 脚本执行超时")
        except json.JSONDecodeError as e:
            logger.error(f"[{name}] 脚本输出不是有效 JSON: {e}")
        except Exception as e:
            logger.error(f"[{name}] 脚本执行失败: {e}")

        return None

    def fetch_single_source(self, source: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        抓取单个源 (自动识别 RSS、JSON API 或脚本)

        Args:
            source: 源配置字典，包含 name, url, category, type

        Returns:
            包含源信息和条目的字典，失败返回 None
        """
        name = source.get('name', 'Unknown')
        source_type = source.get('type', 'rss')

        # 脚本类型不需要 URL
        if source_type != 'script':
            url = source.get('url', '')
            if not url:
                logger.warning(f"[{name}] URL 为空，跳过")
                return None

        if source_type == 'json_api':
            return self._fetch_json_api(source)
        elif source_type == 'script':
            return self._fetch_script(source)
        else:
            # rss, rsshub 等都走 RSS 解析
            return self._fetch_rss(source)

    def fetch_all(self, sources: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        并发抓取所有源

        Args:
            sources: 源配置列表

        Returns:
            成功抓取的结果列表
        """
        results = []

        if not sources:
            logger.warning("没有配置任何源")
            return results

        # 过滤禁用的源
        enabled_sources = [s for s in sources if s.get('enabled', True)]
        disabled_count = len(sources) - len(enabled_sources)

        if disabled_count > 0:
            logger.info(f"跳过 {disabled_count} 个禁用的源")

        if not enabled_sources:
            logger.warning("没有启用的源")
            return results

        logger.info(f"开始并发抓取 {len(enabled_sources)} 个源...")

        with ThreadPoolExecutor(max_workers=self.max_workers) as executor:
            # 提交所有任务
            future_to_source = {
                executor.submit(self.fetch_single_source, source): source
                for source in enabled_sources
            }

            # 收集结果
            for future in as_completed(future_to_source):
                source = future_to_source[future]
                try:
                    result = future.result()
                    if result:
                        results.append(result)
                except Exception as e:
                    logger.error(f"[{source.get('name', 'Unknown')}] 任务执行异常: {e}")

        success_count = len(results)
        fail_count = len(enabled_sources) - success_count
        logger.info(f"抓取完成: 成功 {success_count}, 失败 {fail_count}")

        return results

    def close(self):
        """关闭 session"""
        self.session.close()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()
