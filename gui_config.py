import tkinter as tk
from tkinter import ttk, messagebox, simpledialog
import json
import logging
from pathlib import Path
import sys
import threading
from tkinter.scrolledtext import ScrolledText

# 确保项目根目录在 path 中，以便导入 src
current_dir = Path(__file__).parent.absolute()
sys.path.append(str(current_dir))

try:
    from src.fetcher import RSSFetcher
except ImportError:
    # Fallback for development if not running from root
    RSSFetcher = None

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger("GUI")

CONFIG_FILE = Path("config/sources.json")

# 预定义模版 (从 admin_dashboard.py 移植)
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
    }
}

class ModernTheme:
    """定义暗色系现代主题样式"""
    @staticmethod
    def apply(root):
        style = ttk.Style(root)
        style.theme_use('clam') # 基于 clam 修改

        # 颜色定义
        bg_dark = "#1e222d"
        bg_darker = "#0d1117"
        fg_light = "#e0e0e0"
        accent_blue = "#58a6ff"
        accent_green = "#3fb950"
        border_color = "#30363d"
        select_bg = "#264f78"

        # 配置全局颜色
        root.configure(bg=bg_darker)

        # 配置 TFrame
        style.configure("TFrame", background=bg_darker)
        style.configure("Card.TFrame", background=bg_dark, relief="flat", borderwidth=1)

        # 配置 TLabel
        style.configure("TLabel", background=bg_darker, foreground=fg_light, font=("Segoe UI", 10))
        style.configure("Card.TLabel", background=bg_dark, foreground=fg_light)
        style.configure("Header.TLabel", background=bg_darker, foreground=accent_blue, font=("Segoe UI", 14, "bold"))
        style.configure("Status.TLabel", background=bg_darker, foreground="#8b949e", font=("Segoe UI", 9))

        # 配置 TButton
        style.configure("TButton",
            background=bg_dark,
            foreground=fg_light,
            borderwidth=1,
            bordercolor=border_color,
            focuscolor=accent_blue,
            font=("Segoe UI", 10)
        )
        style.map("TButton",
            background=[("active", "#30363d"), ("pressed", "#21262d")],
            foreground=[("active", "#ffffff")]
        )
        style.configure("Accent.TButton", background=accent_blue, foreground="#ffffff")
        style.map("Accent.TButton", background=[("active", "#79c0ff")])

        # 配置 TEntry
        style.configure("TEntry",
            fieldbackground="#0d1117",
            foreground=fg_light,
            bordercolor=border_color,
            lightcolor=border_color,
            darkcolor=border_color
        )

        # 配置 Treeview
        style.configure("Treeview",
            background=bg_dark,
            fieldbackground=bg_dark,
            foreground=fg_light,
            borderwidth=0,
            rowheight=28,
            font=("Segoe UI", 10)
        )
        style.configure("Treeview.Heading",
            background=bg_darker,
            foreground=fg_light,
            relief="flat",
            font=("Segoe UI", 10, "bold")
        )
        style.map("Treeview", background=[("selected", select_bg)], foreground=[("selected", "#ffffff")])

        # 配置 TNotebook
        style.configure("TNotebook", background=bg_darker, borderwidth=0)
        style.configure("TNotebook.Tab",
            background=bg_darker,
            foreground="#8b949e",
            padding=[10, 5],
            borderwidth=0
        )
        style.map("TNotebook.Tab",
            background=[("selected", bg_dark)],
            foreground=[("selected", fg_light)]
        )

        # 配置 Checkbutton
        style.configure("TCheckbutton", background=bg_dark, foreground=fg_light)

        # Scrollbar
        style.configure("Vertical.TScrollbar", background=bg_dark, troughcolor=bg_darker, borderwidth=0, arrowsize=12)

class NewsPocketGUI:
    def __init__(self, root):
        self.root = root
        self.root.title("NewsPocket 配置管理器")
        self.root.geometry("1000x700")

        # 应用样式
        ModernTheme.apply(root)

        # 数据初始化
        self.config = {"sources": [], "settings": {}}
        self.current_source_index = None
        self.load_config()

        # UI 构建
        self.setup_ui()
        self.refresh_tree()

    def load_config(self):
        if CONFIG_FILE.exists():
            try:
                with open(CONFIG_FILE, 'r', encoding='utf-8') as f:
                    self.config = json.load(f)
            except Exception as e:
                messagebox.showerror("错误", f"无法读取配置文件: {e}")
        else:
            self.config = {"sources": [], "settings": {}}

    def save_config(self):
        try:
            with open(CONFIG_FILE, 'w', encoding='utf-8') as f:
                json.dump(self.config, f, ensure_ascii=False, indent=2)
            # messagebox.showinfo("成功", "配置已保存")
            self.status_var.set("配置已保存到文件 ✅")
        except Exception as e:
            messagebox.showerror("保存失败", str(e))

    def setup_ui(self):
        # 顶部工具栏
        toolbar = ttk.Frame(self.root, padding=10)
        toolbar.pack(fill=tk.X)

        ttk.Label(toolbar, text="NewsPocket Admin", style="Header.TLabel").pack(side=tk.LEFT, padx=(0, 20))

        ttk.Button(toolbar, text="💾 保存所有更改", command=self.save_config, style="Accent.TButton").pack(side=tk.RIGHT, padx=5)
        ttk.Button(toolbar, text="➕ 添加源", command=self.show_add_menu).pack(side=tk.RIGHT, padx=5)
        ttk.Button(toolbar, text="📤 导入 JSON", command=self.show_import_dialog).pack(side=tk.RIGHT, padx=5)

        # 主分割窗口
        self.paned = ttk.PanedWindow(self.root, orient=tk.HORIZONTAL)
        self.paned.pack(fill=tk.BOTH, expand=True, padx=10, pady=(0, 10))

        # 左侧：侧边栏 (Treeview)
        self.setup_sidebar()

        # 右侧：详情页
        self.setup_detail_panel()

        # 底部状态栏
        self.status_var = tk.StringVar(value="就绪")
        ttk.Label(self.root, textvariable=self.status_var, style="Status.TLabel").pack(fill=tk.X, padx=10, pady=5)

    def setup_sidebar(self):
        sidebar_frame = ttk.Frame(self.paned, width=280)
        self.paned.add(sidebar_frame, weight=1)

        # 树形列表
        columns = ("enabled",)
        self.tree = ttk.Treeview(sidebar_frame, columns=columns, show="tree", selectmode="browse")
        self.tree.pack(fill=tk.BOTH, expand=True)

        # 滚动条
        scrollbar = ttk.Scrollbar(sidebar_frame, orient="vertical", command=self.tree.yview)
        scrollbar.pack(side=tk.RIGHT, fill=tk.Y) # 这里有点布局冲突，应该把 tree 和 scrollbar 放一个 frame

        # 修正滚动条布局
        scrollbar.pack_forget()
        self.tree.pack_forget()

        tree_container = ttk.Frame(sidebar_frame)
        tree_container.pack(fill=tk.BOTH, expand=True)

        self.tree.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)
        scrollbar.pack(side=tk.RIGHT, fill=tk.Y)
        self.tree.configure(yscrollcommand=scrollbar.set)

        # 绑定事件
        self.tree.bind("<<TreeviewSelect>>", self.on_tree_select)

        # 删除按钮
        btn_frame = ttk.Frame(sidebar_frame, padding=(0, 5))
        btn_frame.pack(fill=tk.X)
        ttk.Button(btn_frame, text="🗑️ 删除选中", command=self.delete_source).pack(fill=tk.X)

    def setup_detail_panel(self):
        self.detail_frame = ttk.Frame(self.paned, style="Card.TFrame", padding=20)
        self.paned.add(self.detail_frame, weight=3)

        # 详情页容器 (默认隐藏，选中时显示)
        self.content_container = ttk.Frame(self.detail_frame, style="Card.TFrame")
        self.content_container.pack(fill=tk.BOTH, expand=True)

        # --- 基本信息表单 ---

        # 标题栏
        header_frame = ttk.Frame(self.content_container, style="Card.TFrame")
        header_frame.pack(fill=tk.X, pady=(0, 20))
        ttk.Label(header_frame, text="编辑订阅源", style="Header.TLabel").pack(side=tk.LEFT)
        ttk.Button(header_frame, text="🧪 测试抓取", command=self.test_fetch).pack(side=tk.RIGHT)

        # 绑定变量
        self.var_name = tk.StringVar()
        self.var_url = tk.StringVar()
        self.var_category = tk.StringVar()
        self.var_type = tk.StringVar()
        self.var_enabled = tk.BooleanVar()
        self.var_collapsed = tk.BooleanVar()

        # 监听变量变化以自动保存到内存
        for var in [self.var_name, self.var_url, self.var_category, self.var_type]:
            var.trace_add("write", self.on_form_change)
        self.var_enabled.trace_add("write", self.on_form_change)
        self.var_collapsed.trace_add("write", self.on_form_change)

        # 表单布局
        form_grid = ttk.Frame(self.content_container, style="Card.TFrame")
        form_grid.pack(fill=tk.X)

        # Name
        self._create_row(form_grid, 0, "名称:", self.var_name)
        # Category
        self._create_row(form_grid, 1, "分类:", self.var_category) # 后面可以改成 Combobox
        # Type
        self._create_row(form_grid, 2, "类型:", self.var_type, is_combo=True, options=["rss", "json_api", "script"])
        # URL
        self._create_row(form_grid, 3, "URL:", self.var_url)

        # Checkboxes
        check_frame = ttk.Frame(form_grid, style="Card.TFrame")
        check_frame.grid(row=4, column=1, sticky="w", pady=10)
        ttk.Checkbutton(check_frame, text="启用 (Enabled)", variable=self.var_enabled, style="TCheckbutton").pack(side=tk.LEFT, padx=(0, 20))
        ttk.Checkbutton(check_frame, text="默认折叠 (Collapsed)", variable=self.var_collapsed, style="TCheckbutton").pack(side=tk.LEFT)

        # --- 高级配置 (JSON 编辑器) ---
        ttk.Label(self.content_container, text="高级配置 (JSON)", style="Card.TLabel").pack(anchor="w", pady=(20, 5))

        self.json_editor = ScrolledText(self.content_container, height=10, bg="#0d1117", fg="#e0e0e0", insertbackground="white", font=("Consolas", 10))
        self.json_editor.pack(fill=tk.BOTH, expand=True)
        # 绑定 Text 的 KeyRelease 用于保存
        self.json_editor.bind("<KeyRelease>", self.on_json_change)

        # 初始状态：禁用表单
        self.toggle_form(False)

    def _create_row(self, parent, row, label, var, is_combo=False, options=None):
        ttk.Label(parent, text=label, style="Card.TLabel").grid(row=row, column=0, sticky="e", padx=(0, 10), pady=5)
        if is_combo:
            widget = ttk.Combobox(parent, textvariable=var, values=options, state="readonly")
        else:
            widget = ttk.Entry(parent, textvariable=var)
        widget.grid(row=row, column=1, sticky="ew", pady=5)
        parent.columnconfigure(1, weight=1)

    def toggle_form(self, state):
        """启用或禁用右侧表单"""
        status = "normal" if state else "disabled"
        for child in self.content_container.winfo_children():
            # 递归禁用会有问题，简单禁用主要容器即可
            pass
        if not state:
            self.content_container.pack_forget()
            if not hasattr(self, 'placeholder_lbl'):
                self.placeholder_lbl = ttk.Label(self.detail_frame, text="👈 请从左侧选择一个订阅源或添加新源", font=("Segoe UI", 12), foreground="#8b949e")
            self.placeholder_lbl.pack(expand=True)
        else:
            if hasattr(self, 'placeholder_lbl'):
                self.placeholder_lbl.pack_forget()
            self.content_container.pack(fill=tk.BOTH, expand=True)

    def refresh_tree(self):
        """刷新左侧列表"""
        # 保存当前选择
        selected_id = self.tree.selection()

        # 清空
        self.tree.delete(*self.tree.get_children())

        sources = self.config.get("sources", [])

        # 分组
        categories = {}
        for idx, s in enumerate(sources):
            cat = s.get("category", "Uncategorized")
            if cat not in categories:
                categories[cat] = []
            categories[cat].append((idx, s))

        # 填充
        for cat in sorted(categories.keys()):
            cat_node = self.tree.insert("", "end", text=cat, open=True)
            for idx, s in categories[cat]:
                name = s.get("name", "Unnamed")
                enabled = "🟢" if s.get("enabled", True) else "⚪"
                # 使用 iid 存储列表索引
                self.tree.insert(cat_node, "end", iid=str(idx), text=f"{enabled} {name}")

        # 恢复选择
        if selected_id and self.tree.exists(selected_id):
            self.tree.selection_set(selected_id)

    def on_tree_select(self, event):
        selected = self.tree.selection()
        if not selected:
            return

        item_iid = selected[0]
        # 如果选中的是父节点（分类），则不处理
        if self.tree.parent(item_iid) == "":
            self.current_source_index = None
            self.toggle_form(False)
            return

        try:
            idx = int(item_iid)
            self.current_source_index = idx
            self.load_source_to_form(self.config["sources"][idx])
            self.toggle_form(True)
        except ValueError:
            pass

    def load_source_to_form(self, source):
        """将数据加载到右侧表单"""
        # 暂停监听以避免循环触发
        self._ignore_changes = True

        self.var_name.set(source.get("name", ""))
        self.var_url.set(source.get("url", ""))
        self.var_category.set(source.get("category", ""))
        self.var_type.set(source.get("type", "rss"))
        self.var_enabled.set(source.get("enabled", True))
        self.var_collapsed.set(source.get("collapsed", False))

        # 提取高级字段
        advanced = {}
        if "headers" in source:
            advanced["headers"] = source["headers"]
        if "json_config" in source:
            advanced["json_config"] = source["json_config"]
        if "script_config" in source:
            advanced["script_config"] = source["script_config"]

        self.json_editor.delete("1.0", tk.END)
        self.json_editor.insert("1.0", json.dumps(advanced, indent=2, ensure_ascii=False))

        self._ignore_changes = False

    def on_form_change(self, *args):
        """表单字段变更时更新内存数据"""
        if getattr(self, '_ignore_changes', False) or self.current_source_index is None:
            return

        source = self.config["sources"][self.current_source_index]
        source["name"] = self.var_name.get()
        source["url"] = self.var_url.get()
        source["category"] = self.var_category.get()
        source["type"] = self.var_type.get()
        source["enabled"] = self.var_enabled.get()
        source["collapsed"] = self.var_collapsed.get()

        self.status_var.set("有未保存的更改...")
        # 某些更改可能需要刷新列表（如名称、启用状态）
        # 为了性能，可以延迟刷新或只更新特定item，这里简单全刷
        # 但全刷会导致焦点丢失问题，所以这里只在 Save 或特定操作后刷新 Tree UI
        # 或者我们只更新 tree item 的 text

        item_id = str(self.current_source_index)
        if self.tree.exists(item_id):
            enabled_icon = "🟢" if source["enabled"] else "⚪"
            self.tree.item(item_id, text=f"{enabled_icon} {source['name']}")

    def on_json_change(self, event):
        """JSON 编辑器变更时"""
        if getattr(self, '_ignore_changes', False) or self.current_source_index is None:
            return

        text = self.json_editor.get("1.0", tk.END).strip()
        try:
            data = json.loads(text)
            source = self.config["sources"][self.current_source_index]

            # 更新字段
            for key in ["headers", "json_config", "script_config"]:
                if key in data:
                    source[key] = data[key]
                elif key in source:
                    del source[key] # 如果 JSON 中删除了，则源中也删除

            self.json_editor.configure(bg="#0d1117") # 恢复背景
            self.status_var.set("JSON 配置已更新")
        except json.JSONDecodeError:
            self.json_editor.configure(bg="#3d1c1c") # 错误提示背景
            self.status_var.set("JSON 格式错误")

    def show_add_menu(self):
        """显示添加源菜单"""
        menu = tk.Menu(self.root, tearoff=0)
        for name in SOURCE_TEMPLATES.keys():
            menu.add_command(label=name, command=lambda n=name: self.add_source(n))

        # 获取按钮位置
        # 这里简化处理，直接在鼠标位置弹出
        x = self.root.winfo_pointerx()
        y = self.root.winfo_pointery()
        menu.post(x, y)

    def add_source(self, template_name):
        new_source = SOURCE_TEMPLATES[template_name].copy()
        # 避免重名
        base_name = new_source["name"]
        count = 1
        existing_names = [s.get("name") for s in self.config["sources"]]
        while new_source["name"] in existing_names:
            new_source["name"] = f"{base_name} ({count})"
            count += 1

        self.config["sources"].append(new_source)
        self.refresh_tree()
        # 选中新建的
        new_idx = len(self.config["sources"]) - 1
        self.tree.see(str(new_idx))
        self.tree.selection_set(str(new_idx))
        self.status_var.set(f"已添加: {new_source['name']}")

    def delete_source(self):
        selected = self.tree.selection()
        if not selected:
            return

        item_id = selected[0]
        if self.tree.parent(item_id) == "":
            messagebox.showwarning("提示", "请选择具体的源进行删除，不能直接删除分类。")
            return

        idx = int(item_id)
        name = self.config["sources"][idx].get("name")

        if messagebox.askyesno("确认", f"确定要删除 '{name}' 吗?"):
            self.config["sources"].pop(idx)
            self.current_source_index = None
            self.toggle_form(False)
            self.refresh_tree()
            self.status_var.set(f"已删除: {name}")

    def show_import_dialog(self):
        """导入 JSON 对话框"""
        dialog = tk.Toplevel(self.root)
        dialog.title("导入阅读源配置")
        dialog.geometry("600x400")
        ModernTheme.apply(dialog)

        ttk.Label(dialog, text="请粘贴 JSON 配置 (支持阅读APP源格式)", style="Card.TLabel").pack(padx=10, pady=5)
        text_area = ScrolledText(dialog, height=15)
        text_area.pack(fill=tk.BOTH, expand=True, padx=10)

        def do_import():
            content = text_area.get("1.0", tk.END).strip()
            if not content:
                return
            try:
                data = json.loads(content)
                imported_count = 0

                # 逻辑复用 admin_dashboard.py
                if isinstance(data, dict):
                    if "ruleTitle" in data or "sourceUrl" in data:
                        new_item = {
                            "name": data.get("sourceName", "Imported Source"),
                            "url": data.get("sourceUrl", "").replace("{{(page-1)*30}}", ""),
                            "category": data.get("sourceGroup", "Imported"),
                            "type": "json_api",
                            "enabled": True,
                            "collapsed": False,
                            "headers": {"User-Agent": "Mozilla/5.0"},
                            "json_config": {}
                        }
                        if "ruleArticles" in data: new_item["json_config"]["items_path"] = data["ruleArticles"].replace("$.", "")
                        if "ruleTitle" in data: new_item["json_config"]["title_field"] = data["ruleTitle"].replace("$.", "")
                        if "ruleLink" in data: new_item["json_config"]["link_field"] = data["ruleLink"].replace("$.", "")
                        if "ruleDescription" in data: new_item["json_config"]["summary_field"] = data["ruleDescription"].replace("$.", "")

                        self.config["sources"].append(new_item)
                        imported_count = 1
                    elif "url" in data and "name" in data:
                        self.config["sources"].append(data)
                        imported_count = 1
                elif isinstance(data, list):
                    for item in data:
                        if "url" in item and "name" in item:
                            self.config["sources"].append(item)
                            imported_count += 1

                if imported_count > 0:
                    self.refresh_tree()
                    messagebox.showinfo("成功", f"成功导入 {imported_count} 个源")
                    dialog.destroy()
                else:
                    messagebox.showwarning("警告", "未识别到有效的源格式")

            except json.JSONDecodeError:
                messagebox.showerror("错误", "JSON 格式无效")
            except Exception as e:
                messagebox.showerror("错误", str(e))

        ttk.Button(dialog, text="执行导入", command=do_import, style="Accent.TButton").pack(pady=10)

    def test_fetch(self):
        """测试抓取当前源"""
        if self.current_source_index is None:
            return

        source = self.config["sources"][self.current_source_index]

        if not RSSFetcher:
            messagebox.showerror("错误", "无法加载 RSSFetcher 模块，请检查环境。")
            return

        # 创建弹窗显示结果
        result_window = tk.Toplevel(self.root)
        result_window.title(f"测试抓取: {source['name']}")
        result_window.geometry("700x500")
        ModernTheme.apply(result_window)

        log_text = ScrolledText(result_window, bg="#0d1117", fg="#e0e0e0", font=("Consolas", 9))
        log_text.pack(fill=tk.BOTH, expand=True)

        def run_task():
            log_text.insert(tk.END, f"正在连接: {source.get('url')} ...\n")
            try:
                with RSSFetcher(timeout=10) as fetcher:
                    result = fetcher.fetch_single_source(source)

                if result:
                    entries = result.get('entries', [])
                    log_text.insert(tk.END, f"✅ 抓取成功! 发现 {len(entries)} 条内容。\n\n")
                    log_text.insert(tk.END, "-"*50 + "\n")

                    for i, entry in enumerate(entries[:5]): # 只显示前5条
                        title = entry.get('title', 'No Title')
                        link = entry.get('link', 'No Link')
                        log_text.insert(tk.END, f"[{i+1}] {title}\n    {link}\n")

                    if len(entries) > 5:
                        log_text.insert(tk.END, f"\n... 以及其他 {len(entries)-5} 条。\n")

                    log_text.insert(tk.END, "\n[原始数据概览]:\n")
                    # 安全地进行 JSON 序列化
                    try:
                        # 移除不可序列化的对象 (如 feedparser 的 struct_time)
                        raw_preview = str(entries[0]) if entries else "{}"
                        log_text.insert(tk.END, raw_preview[:500] + "...")
                    except:
                        pass
                else:
                    log_text.insert(tk.END, "❌ 抓取失败 (返回为空)。\n")
            except Exception as e:
                log_text.insert(tk.END, f"❌ 发生异常: {e}\n")

        # 异步执行
        threading.Thread(target=run_task, daemon=True).start()

if __name__ == "__main__":
    root = tk.Tk()
    app = NewsPocketGUI(root)
    root.mainloop()
