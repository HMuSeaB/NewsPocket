"""
NewsPocket - Admin Dashboard (Streamlit)
功能强大的可视化配置管理中心
"""

import streamlit as st
import json
import pandas as pd
import sys
import os
import logging
from pathlib import Path
from datetime import datetime
from jinja2 import Template

# 尝试导入 Plotly
try:
    import plotly.express as px
except ImportError:
    px = None

# 1. 基础配置与依赖导入
sys.path.append(os.path.abspath("."))

try:
    from src.fetcher import RSSFetcher
    from src.parser import ContentParser
except ImportError as e:
    st.error(f"无法导入项目模块: {e}. 请确保在项目根目录下运行此脚本。")
    st.stop()

# 常量定义
CONFIG_FILE = Path("config/sources.json")
TEMPLATE_FILE = Path("templates/email_template.html")

# 智能模版 (Smart Templates)
SOURCE_TEMPLATES = {
    "RSS Feed": {
        "type": "rss",
        "name": "New RSS Feed",
        "url": "https://example.com/rss.xml",
        "category": "Tech",
        "enabled": True,
        "collapsed": False
    },
    "JSON API (General)": {
        "type": "json_api",
        "name": "New JSON API",
        "url": "https://api.example.com/data",
        "category": "General",
        "enabled": True,
        "collapsed": False,
        "headers": {"User-Agent": "Mozilla/5.0"},
        "json_config": {
            "items_path": "data.items",
            "title_field": "title",
            "link_field": "url",
            "time_field": "date"
        }
    },
    "Weibo Hot Search": {
        "type": "json_api",
        "name": "微博热搜",
        "url": "https://weibo.com/ajax/side/hotSearch",
        "category": "Social",
        "enabled": True,
        "headers": {"Referer": "https://weibo.com"},
        "json_config": {
            "items_path": "data.realtime",
            "title_field": "word",
            "link_template": "https://s.weibo.com/weibo?q=%23{word}%23"
        }
    },
    "Python Script": {
        "type": "script",
        "name": "Custom Script",
        "category": "Custom",
        "enabled": True,
        "script_config": {
            "type": "python",
            "path": "scripts/example.py",
            "function": "fetch_data"
        }
    }
}

# 国际化翻译字典 (i18n)
TRANSLATIONS = {
    "zh": {
        "page_title": "NewsPocket 管理面板",
        "sidebar_desc": "可视化配置管理中心",
        "filters": "🔍 筛选器",
        "category": "分类",
        "type": "类型",
        "all": "全部",
        "add_source": "➕ 添加新源",
        "choose_template": "选择模版",
        "add_btn": "从模版添加",
        "save_btn": "💾 保存所有更改",
        "config_path": "配置文件:",
        "tab_manage": "📊 管理",
        "tab_test": "🧪 测试",
        "tab_preview": "📧 预览",
        "analytics": "📊 数据统计",
        "chart_cat": "分类分布",
        "chart_type": "类型分布",
        "source_mgmt": "📝 订阅源管理",
        "view_mode": "视图",
        "table": "表格",
        "cards": "卡片",
        "col_enabled": "启用",
        "col_fold": "折叠",
        "col_name": "名称",
        "col_cat": "分类",
        "col_type": "类型",
        "col_url": "链接",
        "no_match": "没有匹配的订阅源。",
        "card_url": "链接:",
        "card_type": "类型:",
        "card_fold": "默认折叠:",
        "switch_tip": "切换到“表格”视图进行编辑。",
        "test_header": "🧪 测试与配置",
        "select_source": "选择订阅源",
        "config_json": "高级配置 (JSON)",
        "run_test": "🚀 运行抓取测试",
        "no_sources": "没有可用的订阅源。",
        "fetching": "正在抓取",
        "fetch_fail": "抓取失败 (无结果)。",
        "fetch_success": "抓取成功!",
        "raw_entries": "原始条目数:",
        "tab_parsed": "解析结果",
        "tab_raw": "原始 JSON",
        "valid_items": "个有效条目。",
        "no_parsed": "未解析出有效条目。",
        "email_header": "📧 邮件预览",
        "gen_preview_desc": "使用所有已启用的源生成完整预览。",
        "gen_btn": "生成完整预览",
        "gen_fetching": "正在抓取所有已启用源...",
        "no_enabled": "没有已启用的订阅源。",
        "tmpl_not_found": "未找到模板文件。",
        "gen_success": "生成成功!",
        "download_html": "下载 HTML",
        "save_success": "配置已保存! ✅",
        "save_fail": "保存失败:",
        "lang_sel": "🌐 Language"
    },
    "en": {
        "page_title": "NewsPocket Admin",
        "sidebar_desc": "Visual Configuration Center",
        "filters": "🔍 Filters",
        "category": "Category",
        "type": "Type",
        "all": "All",
        "add_source": "➕ Add Source",
        "choose_template": "Template",
        "add_btn": "Add from Template",
        "save_btn": "💾 Save All Changes",
        "config_path": "Config:",
        "tab_manage": "📊 Manage",
        "tab_test": "🧪 Test",
        "tab_preview": "📧 Preview",
        "analytics": "📊 Analytics",
        "chart_cat": "Category Distribution",
        "chart_type": "Type Distribution",
        "source_mgmt": "📝 Source Management",
        "view_mode": "View",
        "table": "Table",
        "cards": "Cards",
        "col_enabled": "Enabled",
        "col_fold": "Fold",
        "col_name": "Name",
        "col_cat": "Category",
        "col_type": "Type",
        "col_url": "URL",
        "no_match": "No sources match the filters.",
        "card_url": "URL:",
        "card_type": "Type:",
        "card_fold": "Folded:",
        "switch_tip": "Switch to 'Table' view to edit.",
        "test_header": "🧪 Test & Configure",
        "select_source": "Select Source",
        "config_json": "Configuration (JSON)",
        "run_test": "🚀 Run Fetch Test",
        "no_sources": "No sources available.",
        "fetching": "Fetching",
        "fetch_fail": "Fetch failed (None result).",
        "fetch_success": "Fetch Successful!",
        "raw_entries": "raw entries found.",
        "tab_parsed": "Parsed Result",
        "tab_raw": "Raw JSON",
        "valid_items": "valid items.",
        "no_parsed": "No items parsed.",
        "email_header": "📧 Email Preview",
        "gen_preview_desc": "Generate a full email preview using all enabled sources.",
        "gen_btn": "Generate Full Preview",
        "gen_fetching": "Fetching all enabled sources...",
        "no_enabled": "No enabled sources found.",
        "tmpl_not_found": "Template not found.",
        "gen_success": "Generated!",
        "download_html": "Download HTML",
        "save_success": "Configuration saved successfully! ✅",
        "save_fail": "Save failed:",
        "lang_sel": "🌐 Language"
    }
}

# 设置日志
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("Dashboard")


def t(key):
    """获取当前语言的翻译"""
    lang = st.session_state.get('lang', 'zh') # 默认中文
    return TRANSLATIONS.get(lang, TRANSLATIONS['zh']).get(key, key)


def setup_page():
    """页面初始化与 CSS 注入"""
    st.set_page_config(
        page_title="NewsPocket Admin",
        page_icon="📰",
        layout="wide",
        initial_sidebar_state="expanded"
    )

    # 注入自定义 CSS (Cinematic Dusk Style)
    st.markdown("""
    <style>
        /* 全局背景：深冷蓝暮光渐变 */
        .stApp {
            background: linear-gradient(135deg, #0d1117 0%, #1a2a4a 100%);
            color: #e0e0e0;
            font-family: 'Inter', sans-serif;
        }

        /* 侧边栏：半透明磨砂玻璃 */
        section[data-testid="stSidebar"] {
            background-color: rgba(22, 27, 34, 0.85);
            backdrop-filter: blur(12px);
            border-right: 1px solid rgba(255, 255, 255, 0.08);
        }

        /* 标题：微光效果 */
        h1, h2, h3 {
            color: #ffffff !important;
            text-shadow: 0 0 20px rgba(0, 242, 255, 0.3);
            font-weight: 700;
            letter-spacing: 0.5px;
        }

        /* 卡片容器：毛玻璃与边框光效 */
        div[data-testid="stMetric"], div[data-testid="stExpander"], [data-testid="stDataFrame"] {
            background: rgba(30, 34, 45, 0.6) !important;
            border: 1px solid rgba(255, 255, 255, 0.1) !important;
            border-radius: 12px !important;
            backdrop-filter: blur(10px);
            box-shadow: 0 4px 24px -1px rgba(0, 0, 0, 0.2);
            transition: transform 0.2s ease, box-shadow 0.2s ease;
        }

        div[data-testid="stMetric"]:hover, div[data-testid="stExpander"]:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 32px rgba(0, 242, 255, 0.15);
            border-color: rgba(0, 242, 255, 0.3) !important;
        }

        /* Metric 值：霓虹橙/青 */
        div[data-testid="stMetricValue"] {
            color: #00f2ff;
            text-shadow: 0 0 10px rgba(0, 242, 255, 0.4);
        }
        div[data-testid="stMetricLabel"] {
            color: #a0a0b0;
        }

        /* 按钮：暮光渐变 (橙->粉) */
        button[kind="primary"] {
            background: linear-gradient(90deg, #ff6b4a 0%, #f4a0a0 100%) !important;
            border: none !important;
            color: #1a1a2e !important;
            font-weight: 700 !important;
            box-shadow: 0 0 15px rgba(255, 107, 74, 0.4);
            transition: all 0.3s ease;
        }
        button[kind="primary"]:hover {
            box-shadow: 0 0 25px rgba(255, 107, 74, 0.6);
            transform: scale(1.02);
        }

        /* 输入框：深色半透明 */
        .stTextInput input, .stSelectbox div[data-baseweb="select"] > div, .stTextArea textarea {
            background-color: rgba(0, 0, 0, 0.3) !important;
            color: #ffffff !important;
            border: 1px solid rgba(255, 255, 255, 0.1) !important;
            border-radius: 8px;
        }

        /* 滚动条美化 */
        ::-webkit-scrollbar {
            width: 8px;
            height: 8px;
        }
        ::-webkit-scrollbar-track {
            background: #0d1117;
        }
        ::-webkit-scrollbar-thumb {
            background: #30363d;
            border-radius: 4px;
        }
        ::-webkit-scrollbar-thumb:hover {
            background: #58a6ff;
        }
    </style>
    """, unsafe_allow_html=True)


def load_config():
    """读取配置文件"""
    if not CONFIG_FILE.exists():
        st.error(f"配置文件未找到: {CONFIG_FILE}")
        return {"sources": [], "settings": {}}

    try:
        with open(CONFIG_FILE, 'r', encoding='utf-8') as f:
            return json.load(f)
    except Exception as e:
        st.error(f"读取配置文件失败: {e}")
        return {"sources": [], "settings": {}}


def save_config(config_data):
    """保存配置文件"""
    try:
        with open(CONFIG_FILE, 'w', encoding='utf-8') as f:
            json.dump(config_data, f, ensure_ascii=False, indent=2)
        st.toast(t('save_success'), icon="💾")
    except Exception as e:
        st.error(f"{t('save_fail')} {e}")


def render_sidebar(config):
    """渲染侧边栏"""
    with st.sidebar:
        st.title(t('page_title'))

        # 语言选择
        lang_choice = st.radio(t('lang_sel'), ["中文", "English"], horizontal=True, label_visibility="collapsed")
        st.session_state['lang'] = 'zh' if lang_choice == "中文" else 'en'

        st.markdown(t('sidebar_desc'))
        st.divider()

        # 筛选器
        st.subheader(t('filters'))
        sources_list = config.get("sources", [])

        all_categories = sorted(list(set(s.get("category", "") for s in sources_list))) if sources_list else []
        all_types = sorted(list(set(s.get("type", "") for s in sources_list))) if sources_list else []

        filters = {
            "category": st.selectbox(t('category'), [t('all')] + all_categories),
            "type": st.selectbox(t('type'), [t('all')] + all_types)
        }

        st.divider()

        # 模版添加功能
        st.subheader(t('add_source'))
        selected_template = st.selectbox(t('choose_template'), list(SOURCE_TEMPLATES.keys()))
        if st.button(t('add_btn')):
            new_source = SOURCE_TEMPLATES[selected_template].copy()
            # 避免重名
            base_name = new_source["name"]
            count = 1
            while any(s["name"] == new_source["name"] for s in config["sources"]):
                new_source["name"] = f"{base_name} ({count})"
                count += 1

            config["sources"].append(new_source)
            save_config(config)
            st.rerun()

        st.divider()

        # 全局保存
        if st.button(t('save_btn'), type="primary", use_container_width=True):
            save_config(config)

        st.caption(f"{t('config_path')} `{CONFIG_FILE}`")

    return filters, all_categories


def render_analytics(sources):
    """渲染分析图表 (Cinematic Style)"""
    if not px or not sources:
        return

    # 1. 关键指标 (Metrics)
    total = len(sources)
    enabled = len([s for s in sources if s.get('enabled')])
    disabled = total - enabled

    st.markdown("### 📈 核心指标")
    m1, m2, m3 = st.columns(3)
    m1.metric("总订阅源", total, delta="Sources")
    m2.metric("活跃中", enabled, delta="Running", delta_color="normal") # normal=green
    m3.metric("已暂停", disabled, delta="Paused", delta_color="off")   # off=gray

    st.markdown("---")

    # 2. 视觉图表 (Charts)
    st.subheader(t('analytics'))
    viz_df = pd.DataFrame(sources)

    # 暮光配色方案 (Twilight/Neon Palette)
    # 对应: 青, 橙, 紫, 粉, 蓝, 绿
    colors = ['#00f2ff', '#ff6b4a', '#a371f7', '#f4a0a0', '#58a6ff', '#3fb950']

    col1, col2 = st.columns(2)

    common_layout = dict(
        paper_bgcolor="rgba(0,0,0,0)",
        plot_bgcolor="rgba(0,0,0,0)",
        font=dict(family="Inter, sans-serif", color="#e0e0e0", size=14),
        showlegend=True,
        legend=dict(orientation="h", yanchor="bottom", y=-0.3, xanchor="center", x=0.5), # 下移图例
        margin=dict(t=30, b=60, l=40, r=40) # 增加左右下边距，防止标签截断
    )

    with col1:
        if "category" in viz_df.columns:
            cat_counts = viz_df["category"].value_counts().reset_index()
            cat_counts.columns = ["Category", "Count"]

            fig = px.pie(
                cat_counts,
                values="Count",
                names="Category",
                title=t('chart_cat'),
                hole=0.6, # 更大的中空区域 (Donut)
                color_discrete_sequence=colors
            )
            fig.update_traces(
                textinfo='percent',
                textposition='outside',
                marker=dict(line=dict(color='#1a2a4a', width=3)), # 深蓝描边
                pull=[0.05] * len(cat_counts) # 轻微炸开效果
            )
            fig.update_layout(**common_layout)
            st.plotly_chart(fig, use_container_width=True)

    with col2:
        if "type" in viz_df.columns:
            type_counts = viz_df["type"].value_counts().reset_index()
            type_counts.columns = ["Type", "Count"]

            fig = px.pie(
                type_counts,
                values="Count",
                names="Type",
                title=t('chart_type'),
                hole=0.6,
                color_discrete_sequence=colors[::-1] # 反转配色以区分
            )
            fig.update_traces(
                textinfo='percent',
                textposition='outside',
                marker=dict(line=dict(color='#1a2a4a', width=3)),
                pull=[0.05] * len(type_counts)
            )
            fig.update_layout(**common_layout)
            st.plotly_chart(fig, use_container_width=True)

    st.divider()


def render_source_manager(config, filters, all_categories):
    """渲染源管理器 (表格/卡片)"""
    sources_data = config.get("sources", [])

    # 视图切换
    col_title, col_view = st.columns([6, 1])
    with col_title:
        st.header(t('source_mgmt'))
    with col_view:
        view_mode = st.radio(t('view_mode'), [t('table'), t('cards')], horizontal=True, label_visibility="collapsed")

    # 准备数据 & 筛选
    display_rows = []
    for idx, source in enumerate(sources_data):
        if filters["category"] != t('all') and source.get("category") != filters["category"]:
            continue
        if filters["type"] != t('all') and source.get("type") != filters["type"]:
            continue

        row = source.copy()
        row["_source_index"] = idx  # Track original index
        # Ensure fields exist
        for field in ["enabled", "collapsed", "name", "category", "type", "url"]:
            if field not in row:
                row[field] = "" if field != "enabled" and field != "collapsed" else False
        display_rows.append(row)

    if not display_rows:
        st.info(t('no_match'))
        return

    if view_mode == t('table'):
        df = pd.DataFrame(display_rows)

        column_config = {
            "_source_index": None,
            "headers": None, "json_config": None, "script_config": None, "comment": None, # Hide complex fields
            "enabled": st.column_config.CheckboxColumn(t('col_enabled'), width="small"),
            "collapsed": st.column_config.CheckboxColumn(t('col_fold'), width="small"),
            "name": st.column_config.TextColumn(t('col_name'), width="medium", required=True),
            "category": st.column_config.SelectboxColumn(t('col_cat'), options=all_categories + ["New..."], width="medium", required=True),
            "type": st.column_config.SelectboxColumn(t('col_type'), options=["rss", "json_api", "script"], width="small", required=True),
            "url": st.column_config.TextColumn(t('col_url'), width="large")
        }

        # 只显示核心列
        cols_to_show = ["_source_index", "enabled", "collapsed", "name", "category", "type", "url"]

        edited_df = st.data_editor(
            df[cols_to_show],
            column_config=column_config,
            use_container_width=True,
            num_rows="dynamic",
            key="source_editor",
            hide_index=True
        )

        # 实时更新配置 (Handle Add/Update/Delete)
        if not edited_df.equals(df[cols_to_show]):
            new_list = []

            # Reconstruct list from edited dataframe
            for _, row in edited_df.iterrows():
                # 1. Existing Source (Keep hidden fields)
                if pd.notna(row.get("_source_index")):
                    idx = int(row["_source_index"])
                    if 0 <= idx < len(sources_data):
                        # Get original object
                        source_obj = sources_data[idx]
                        # Update visible fields
                        for field in ["enabled", "collapsed", "name", "category", "type", "url"]:
                            if field in row:
                                source_obj[field] = row[field]
                        new_list.append(source_obj)
                # 2. New Source
                else:
                    new_source = {k: row[k] for k in ["enabled", "collapsed", "name", "category", "type", "url"] if k in row}
                    # Init defaults
                    new_source.setdefault("headers", {})
                    new_source.setdefault("json_config", {})
                    new_list.append(new_source)

            # Apply changes to memory
            config["sources"] = new_list

    else:
        # Card View
        df_view = pd.DataFrame(display_rows)
        grouped = df_view.groupby("category")

        for category, group in grouped:
            st.markdown(f"#### 📂 {category}")
            cols = st.columns(2)
            for i, (_, row) in enumerate(group.iterrows()):
                col = cols[i % 2]
                with col:
                    status = "🟢" if row['enabled'] else "⚪"
                    fold = "📂" if row.get('collapsed', False) else "📖"
                    with st.expander(f"{status} {row['name']} {fold}"):
                        st.markdown(f"**{t('card_url')}** `{row['url']}`")
                        st.caption(f"{t('card_type')} {row['type']}")

                        # Actions
                        c_edit, c_del = st.columns([3, 1])
                        with c_edit:
                            if st.button("⚙️ Config", key=f"btn_edit_{row['_source_index']}", use_container_width=True):
                                st.session_state["selected_source_index"] = int(row["_source_index"])
                                st.info(t('switch_tip'))
                        with c_del:
                            if st.button("🗑️", key=f"btn_del_{row['_source_index']}", type="primary", use_container_width=True):
                                # Delete logic
                                idx_to_del = int(row["_source_index"])
                                if 0 <= idx_to_del < len(config["sources"]):
                                    config["sources"].pop(idx_to_del)
                                    st.rerun()


def render_test_playground(config):
    """渲染测试实验室"""
    st.header(t('test_header'))

    col_sel, col_action = st.columns([1, 2])

    sources = config.get("sources", [])
    source_names = [f"{i}: {s.get('name', 'Unnamed')}" for i, s in enumerate(sources)]

    # 尝试恢复之前的选择
    default_idx = 0
    if "selected_source_index" in st.session_state:
        idx = st.session_state["selected_source_index"]
        if 0 <= idx < len(sources):
            default_idx = idx

    with col_sel:
        selected_idx = st.selectbox(t('select_source'), range(len(sources)),
                                  format_func=lambda x: source_names[x] if x < len(source_names) else "",
                                  index=default_idx)

        if sources:
            current_source = sources[selected_idx]
            st.session_state["selected_source_index"] = selected_idx

            # JSON 编辑器
            st.subheader(t('config_json'))
            source_json = json.dumps(current_source, indent=2, ensure_ascii=False)
            new_json = st.text_area("Edit Config", value=source_json, height=400)

            try:
                new_source_obj = json.loads(new_json)
                if new_source_obj != current_source:
                    config["sources"][selected_idx] = new_source_obj
                    st.toast("Config updated in memory. Save to persist!", icon="⚠️")
            except json.JSONDecodeError:
                st.error("Invalid JSON")

    with col_action:
        st.subheader(t('run_test'))
        if st.button(t('run_test'), type="primary"):
            if not sources:
                st.warning(t('no_sources'))
            else:
                target = sources[selected_idx]
                with st.spinner(f"{t('fetching')} '{target.get('name')}'..."):
                    try:
                        with RSSFetcher(timeout=10) as fetcher:
                            result = fetcher.fetch_single_source(target)

                        if not result:
                            st.error(t('fetch_fail'))
                        else:
                            st.success(t('fetch_success'))

                            parser = ContentParser()
                            try:
                                parsed_items = parser.parse_all([result])
                            except:
                                parsed_items = []

                            st.markdown(f"**Found {len(result.get('entries', []))} {t('raw_entries')}**")

                            tab_p, tab_r = st.tabs([t('tab_parsed'), t('tab_raw')])
                            with tab_p:
                                if parsed_items:
                                    st.info(f"Parsed {len(parsed_items)} {t('valid_items')}")
                                    for item in parsed_items:
                                        with st.expander(f"{item.get('title')}"):
                                            st.write(item)
                                else:
                                    st.warning(t('no_parsed'))
                            with tab_r:
                                st.json(result)
                    except Exception as e:
                        st.error(f"Error: {e}")


def render_email_preview(config):
    """渲染邮件预览"""
    st.header(t('email_header'))

    st.markdown(t('gen_preview_desc'))

    if st.button(t('gen_btn')):
        with st.spinner(t('gen_fetching')):
            try:
                enabled = [s for s in config.get("sources", []) if s.get("enabled")]
                if not enabled:
                    st.warning(t('no_enabled'))
                    return

                source_configs = {s.get('name'): s for s in enabled}

                with RSSFetcher(max_workers=5) as fetcher:
                    results = fetcher.fetch_all(enabled)

                parser = ContentParser()
                items = parser.parse_all(results)
                news_by_cat = parser.group_by_category(items)

                if not TEMPLATE_FILE.exists():
                    st.error(t('tmpl_not_found'))
                    return

                with open(TEMPLATE_FILE, 'r', encoding='utf-8') as f:
                    template = Template(f.read())

                stats = {
                    "total_count": len(items),
                    "source_count": len(set(i['source'] for i in items)),
                    "category_count": len(news_by_cat)
                }

                html = template.render(
                    title=f"NewsPocket Preview",
                    date=datetime.now().strftime('%Y-%m-%d'),
                    news_by_category=news_by_cat,
                    source_configs=source_configs,
                    **stats
                )

                st.success(f"{t('gen_success')} {len(items)} items.")
                st.download_button(t('download_html'), html, "preview.html", "text/html")
                st.components.v1.html(html, height=800, scrolling=True)

            except Exception as e:
                st.error(f"Preview failed: {e}")


def main():
    setup_page()

    if 'config' not in st.session_state:
        st.session_state.config = load_config()

    # 默认语言设置
    if 'lang' not in st.session_state:
        st.session_state['lang'] = 'zh'

    config = st.session_state.config

    filters, categories = render_sidebar(config)

    tab_manage, tab_test, tab_preview = st.tabs([t('tab_manage'), t('tab_test'), t('tab_preview')])

    with tab_manage:
        render_analytics(config.get("sources"))
        render_source_manager(config, filters, categories)

    with tab_test:
        render_test_playground(config)

    with tab_preview:
        render_email_preview(config)


if __name__ == "__main__":
    main()
