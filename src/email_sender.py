"""
NewsPocket - Email Sender Module
渲染 HTML 模板并发送邮件
"""

import logging
import smtplib
import os
from datetime import datetime, timedelta, timezone
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
from email.utils import formataddr
from typing import Dict, List, Any

from jinja2 import Environment, FileSystemLoader

logger = logging.getLogger(__name__)


class EmailSender:
    """邮件发送器"""

    def __init__(self, template_dir: str = 'templates'):
        """
        初始化发送器

        Args:
            template_dir: 模板目录路径
        """
        self.env = Environment(loader=FileSystemLoader(template_dir))
        self.user = os.environ.get('EMAIL_USER')
        self.password = os.environ.get('EMAIL_PASS')
        self.host = os.environ.get('EMAIL_HOST', 'smtp.qq.com')
        self.port = int(os.environ.get('EMAIL_PORT', 465))

        # 接收者可以是单个字符串或以逗号分隔的字符串
        recipient_env = os.environ.get('EMAIL_TO')
        if recipient_env:
            self.recipients = [r.strip() for r in recipient_env.split(',') if r.strip()]
        else:
            self.recipients = [self.user] if self.user else []

    def render_template(self, template_name: str, context: Dict[str, Any]) -> str:
        """
        渲染 Jinja2 模板

        Args:
            template_name: 模板文件名
            context: 上下文数据

        Returns:
            渲染后的 HTML 字符串
        """
        try:
            template = self.env.get_template(template_name)
            return template.render(**context)
        except Exception as e:
            logger.error(f"模板渲染失败: {e}")
            raise

    def send_email(self, subject: str, html_content: str):
        """
        发送 HTML 邮件

        Args:
            subject: 邮件主题
            html_content: HTML 内容
        """
        if not self.user or not self.password:
            logger.error("未配置 EMAIL_USER 或 EMAIL_PASS 环境变量")
            return

        if not self.recipients:
            logger.error("未配置接收者邮箱 (EMAIL_TO)")
            return

        try:
            # 创建邮件对象
            msg = MIMEMultipart()
            msg['From'] = formataddr(("NewsPocket", self.user))
            msg['To'] = ",".join(self.recipients)
            msg['Subject'] = subject

            # 添加 HTML 内容
            msg.attach(MIMEText(html_content, 'html', 'utf-8'))

            logger.info(f"连接 SMTP 服务器: {self.host}:{self.port}")

            # 连接 SMTP 服务器 (使用 SSL)
            with smtplib.SMTP_SSL(self.host, self.port) as server:
                server.login(self.user, self.password)
                server.sendmail(self.user, self.recipients, msg.as_string())

            logger.info(f"邮件已成功发送给: {self.recipients}")

        except Exception as e:
            logger.error(f"邮件发送失败: {e}")
            raise

    def send_daily_news(self, news_by_category: Dict[str, List[Any]], stats: Dict[str, int], source_configs: Dict[str, Any] = None):
        """
        发送每日新闻简报

        Args:
            news_by_category: 分类后的新闻列表
            stats: 统计信息
            source_configs: 源配置字典 (name -> config)
        """
        # 获取当前日期 (北京时间)
        beijing_tz = timezone(timedelta(hours=8))
        today = datetime.now(beijing_tz).strftime('%Y年%m月%d日 %A')

        context = {
            'title': f'NewsPocket 晨报 - {today}',
            'date': today,
            'news_by_category': news_by_category,
            'total_count': stats.get('total', 0),
            'source_count': stats.get('sources', 0),
            'category_count': stats.get('categories', 0),
            'source_configs': source_configs or {}
        }

        logger.info("正在渲染邮件模板...")
        html_content = self.render_template('email_template.html', context)

        logger.info("正在发送邮件...")
        self.send_email(f"NewsPocket 每日简报 - {today}", html_content)
