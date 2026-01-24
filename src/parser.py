"""
NewsPocket - Parser Module
解析和清洗 RSS 内容，输出统一格式
"""

import re
import html
import logging
from datetime import datetime, timedelta, timezone
from typing import List, Dict, Any, Optional
from email.utils import parsedate_to_datetime

from dateutil import parser as date_parser

logger = logging.getLogger(__name__)


class ContentParser:
    """RSS 内容解析器"""

    # 需要移除的 HTML 标签模式
    HTML_TAG_PATTERN = re.compile(r'<[^>]+>')
    # 多余空白字符
    WHITESPACE_PATTERN = re.compile(r'\s+')
    # 常见的无意义内容
    NOISE_PATTERNS = [
        re.compile(r'\[图片\]'),
        re.compile(r'\[视频\]'),
        re.compile(r'点击查看.*'),
        re.compile(r'阅读原文.*'),
        re.compile(r'展开全文.*'),
    ]

    def __init__(self, max_summary_length: int = 200, hours_lookback: int = 24):
        """
        初始化解析器

        Args:
            max_summary_length: 摘要最大长度
            hours_lookback: 只获取多少小时内的内容
        """
        self.max_summary_length = max_summary_length
        self.hours_lookback = hours_lookback
        self.cutoff_time = datetime.now(timezone.utc) - timedelta(hours=hours_lookback)

    def clean_html(self, text: str) -> str:
        """
        清洗 HTML 标签和特殊字符

        Args:
            text: 原始文本

        Returns:
            清洗后的纯文本
        """
        if not text:
            return ""

        # 解码 HTML 实体
        text = html.unescape(text)

        # 移除 HTML 标签
        text = self.HTML_TAG_PATTERN.sub(' ', text)

        # 移除噪音内容
        for pattern in self.NOISE_PATTERNS:
            text = pattern.sub('', text)

        # 规范化空白字符
        text = self.WHITESPACE_PATTERN.sub(' ', text)

        return text.strip()

    def truncate_summary(self, text: str) -> str:
        """
        截断摘要到指定长度

        Args:
            text: 原始摘要

        Returns:
            截断后的摘要
        """
        if not text:
            return ""

        if len(text) <= self.max_summary_length:
            return text

        # 尝试在句子边界截断
        truncated = text[:self.max_summary_length]

        # 寻找最后一个句子结束符
        for sep in ['。', '！', '？', '.', '!', '?', '；', ';']:
            last_sep = truncated.rfind(sep)
            if last_sep > self.max_summary_length * 0.6:  # 至少保留60%
                return truncated[:last_sep + 1]

        # 没有合适的句子边界，直接截断并加省略号
        return truncated.rstrip() + '...'

    def parse_time(self, entry: Dict[str, Any]) -> Optional[datetime]:
        """
        解析条目的发布时间

        Args:
            entry: RSS 条目

        Returns:
            datetime 对象，解析失败返回 None
        """
        # 尝试多个时间字段
        time_fields = ['published', 'updated', 'created', 'pubDate']

        for field in time_fields:
            time_str = entry.get(field) or entry.get(f'{field}_parsed')

            if time_str:
                try:
                    # feedparser 已解析的时间结构
                    if isinstance(time_str, tuple):
                        return datetime(*time_str[:6], tzinfo=timezone.utc)

                    # 尝试标准解析
                    parsed = date_parser.parse(str(time_str))

                    # 确保有时区信息
                    if parsed.tzinfo is None:
                        parsed = parsed.replace(tzinfo=timezone.utc)

                    return parsed

                except Exception:
                    continue

        return None

    def format_time(self, dt: Optional[datetime]) -> str:
        """
        格式化时间为统一格式

        Args:
            dt: datetime 对象

        Returns:
            格式化的时间字符串
        """
        if not dt:
            return "未知时间"

        # 转换为北京时间 (UTC+8)
        beijing_tz = timezone(timedelta(hours=8))
        local_time = dt.astimezone(beijing_tz)

        return local_time.strftime('%Y-%m-%d %H:%M')

    def is_recent(self, dt: Optional[datetime]) -> bool:
        """
        检查时间是否在有效范围内

        Args:
            dt: datetime 对象

        Returns:
            是否是最近的内容
        """
        if not dt:
            # 没有时间信息的条目默认包含
            return True

        return dt >= self.cutoff_time

    def parse_entry(self, entry: Dict[str, Any], source_info: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        解析单个条目 (支持 RSS 和 JSON API)

        Args:
            entry: RSS 条目或 JSON API 条目
            source_info: 源信息

        Returns:
            统一格式的条目字典
        """
        is_json_api = entry.get('_source_type') == 'json_api'

        # 解析时间
        pub_time = self.parse_time(entry)

        # 过滤非最近的内容 (JSON API 可能没有时间字段，放宽限制)
        if not is_json_api and not self.is_recent(pub_time):
            return None

        # 获取标题
        title = self.clean_html(entry.get('title', ''))
        if not title:
            return None

        # 获取摘要
        summary = ''
        if is_json_api:
            # JSON API 直接取 summary 字段
            summary = self.clean_html(entry.get('summary', ''))
        else:
            # RSS: 优先 summary，其次 description，最后 content
            for field in ['summary', 'description']:
                if entry.get(field):
                    summary = self.clean_html(entry.get(field, ''))
                    break

            # 尝试从 content 获取
            if not summary and entry.get('content'):
                content_list = entry.get('content', [])
                if content_list and isinstance(content_list, list):
                    summary = self.clean_html(content_list[0].get('value', ''))

        summary = self.truncate_summary(summary)

        # 获取链接
        link = entry.get('link', '')
        if not link and entry.get('links'):
            links = entry.get('links', [])
            if links:
                link = links[0].get('href', '')

        return {
            'title': title,
            'time': self.format_time(pub_time),
            'time_obj': pub_time,  # 保留原始时间用于排序
            'summary': summary,
            'link': link,
            'source': source_info.get('name', 'Unknown'),
            'category': source_info.get('category', '其他')
        }

    def parse_feed_result(self, fetch_result: Dict[str, Any], max_items: int = 5) -> List[Dict[str, Any]]:
        """
        解析单个源的抓取结果

        Args:
            fetch_result: fetcher 返回的结果
            max_items: 每个源最多保留的条目数

        Returns:
            解析后的条目列表
        """
        source_info = fetch_result.get('source', {})
        entries = fetch_result.get('entries', [])

        parsed_items = []

        for entry in entries:
            try:
                parsed = self.parse_entry(entry, source_info)
                if parsed:
                    parsed_items.append(parsed)
            except Exception as e:
                logger.warning(f"解析条目失败: {e}")
                continue

        # 按时间排序（最新在前）
        parsed_items.sort(
            key=lambda x: x.get('time_obj') or datetime.min.replace(tzinfo=timezone.utc),
            reverse=True
        )

        # 限制数量
        return parsed_items[:max_items]

    def parse_all(self, fetch_results: List[Dict[str, Any]], max_items_per_source: int = 5) -> List[Dict[str, Any]]:
        """
        解析所有抓取结果

        Args:
            fetch_results: fetcher 返回的所有结果
            max_items_per_source: 每个源最多保留的条目数

        Returns:
            所有解析后的条目列表
        """
        all_items = []

        for result in fetch_results:
            items = self.parse_feed_result(result, max_items_per_source)
            all_items.extend(items)
            logger.info(f"[{result.get('source', {}).get('name', 'Unknown')}] 解析出 {len(items)} 条有效内容")

        # 全局按时间排序
        all_items.sort(
            key=lambda x: x.get('time_obj') or datetime.min.replace(tzinfo=timezone.utc),
            reverse=True
        )

        logger.info(f"共解析出 {len(all_items)} 条有效内容")

        return all_items

    def group_by_category(self, items: List[Dict[str, Any]]) -> Dict[str, List[Dict[str, Any]]]:
        """
        按分类分组

        Args:
            items: 条目列表

        Returns:
            分类字典
        """
        grouped = {}

        # 预定义分类顺序
        category_order = ['行业动态', '全球热点', '科技生活', '社交热点', '其他']

        # 初始化所有分类
        for cat in category_order:
            grouped[cat] = []

        # 分组
        for item in items:
            category = item.get('category', '其他')
            if category not in grouped:
                grouped[category] = []
            grouped[category].append(item)

        # 移除空分类（保持预定义顺序）
        result = {}
        for cat in category_order:
            if cat in grouped and grouped[cat]:
                result[cat] = grouped[cat]

        # 添加其他未预定义的分类
        for cat, items_list in grouped.items():
            if cat not in result and items_list:
                result[cat] = items_list

        return result
