let currentConfig = { sources: [], settings: {} };
let activeSourceIndex = -1;
let searchQuery = '';

const DOM = {
    sourceList: document.getElementById('source-list'),
    form: document.getElementById('source-form'),
    emptyState: document.getElementById('empty-state'),
    jsonSection: document.getElementById('json-config-section'),
    headersContainer: document.getElementById('headers-container'),
    saveStatus: document.getElementById('save-status'),
    searchSources: document.getElementById('search-sources'),

    // Global Buttons
    btnOpenConfig: document.getElementById('btn-open-config'),
    btnSave: document.getElementById('btn-save'),
    btnAddRss: document.getElementById('btn-add-rss'),
    btnAddJson: document.getElementById('btn-add-json'),
    btnImportOpml: document.getElementById('btn-import-opml'),
    btnExportOpml: document.getElementById('btn-export-opml'),
    btnPreviewEmail: document.getElementById('btn-preview-email'),
    emptyBtnPreview: document.getElementById('empty-btn-preview'),
    btnAddHeader: document.getElementById('btn-add-header'),
    btnFetchTest: document.getElementById('btn-fetch-test'),

    // Modals
    testModal: document.getElementById('test-result-modal'),
    testResultText: document.getElementById('test-result-text'),
    btnCloseTestModal: document.getElementById('btn-close-test-modal'),

    previewModal: document.getElementById('preview-modal'),
    previewFrame: document.getElementById('preview-frame'),
    previewLoading: document.getElementById('preview-loading'),
    btnRefreshPreview: document.getElementById('btn-refresh-preview'),
    btnClosePreviewModal: document.getElementById('btn-close-preview-modal'),

    // Form fields
    fName: document.getElementById('name'),
    fCategory: document.getElementById('category'),
    fUrl: document.getElementById('url'),
    fEnabled: document.getElementById('enabled'),
    fType: document.getElementById('type'),
    fjItemsPath: document.getElementById('items_path'),
    fjTitle: document.getElementById('title_field'),
    fjSummary: document.getElementById('summary_field'),
    fjTime: document.getElementById('time_field'),
    fjLink: document.getElementById('link_field'),
    fjLinkTpl: document.getElementById('link_template')
};

// Initialize app
async function init() {
    // 绑定事件
    DOM.btnAddRss.onclick = () => createNewSource('rss');
    DOM.btnAddJson.onclick = () => createNewSource('json_api');
    DOM.btnImportOpml.onclick = importOPML;
    DOM.btnExportOpml.onclick = exportOPML;
    DOM.btnSave.onclick = saveConfig;
    DOM.btnOpenConfig.onclick = openLocalConfig;
    DOM.btnFetchTest.onclick = testFetch;
    DOM.btnPreviewEmail.onclick = openEmailPreview;
    if (DOM.emptyBtnPreview) DOM.emptyBtnPreview.onclick = openEmailPreview;
    DOM.btnRefreshPreview.onclick = openEmailPreview;
    DOM.btnAddHeader.onclick = () => addHeaderRow('', '');

    // 模态框关闭
    DOM.btnCloseTestModal.onclick = () => { DOM.testModal.style.display = 'none'; };
    DOM.btnClosePreviewModal.onclick = () => { DOM.previewModal.style.display = 'none'; };

    // 搜索过滤监听
    DOM.searchSources.oninput = (e) => {
        searchQuery = e.target.value.toLowerCase().trim();
        renderSourceList();
    };

    setupRealtimeSync();

    try {
        await reloadConfig();
    } catch (e) {
        showStatus('加载配置失败: 请点击「打开配置文件」', true);
    }
}

async function reloadConfig() {
    const configStr = await window.go.main.App.GetConfig();
    currentConfig = JSON.parse(configStr);
    if (!currentConfig.sources) currentConfig.sources = [];
    renderSourceList();
}

function renderSourceList() {
    DOM.sourceList.innerHTML = '';

    const sources = currentConfig.sources || [];
    let matchedCount = 0;

    sources.forEach((src, idx) => {
        // 搜索过滤
        if (searchQuery) {
            const name = (src.name || '').toLowerCase();
            const category = (src.category || '').toLowerCase();
            const url = (src.url || '').toLowerCase();
            if (!name.includes(searchQuery) && !category.includes(searchQuery) && !url.includes(searchQuery)) {
                return;
            }
        }
        matchedCount++;

        const li = document.createElement('li');
        li.className = `source-item ${idx === activeSourceIndex ? 'active' : ''} ${src.enabled === false ? 'disabled' : ''}`;

        // 头部行：开关 + 标题
        const headerDiv = document.createElement('div');
        headerDiv.className = 'item-main-row';

        // 快速启停开关
        const toggle = document.createElement('input');
        toggle.type = 'checkbox';
        toggle.className = 'item-toggle';
        toggle.checked = src.enabled !== false;
        toggle.title = src.enabled !== false ? '点击停用' : '点击启用';
        toggle.onclick = (e) => {
            e.stopPropagation();
            src.enabled = toggle.checked;
            if (activeSourceIndex === idx) {
                DOM.fEnabled.checked = src.enabled;
            }
            renderSourceList();
        };

        const titleSpan = document.createElement('span');
        titleSpan.className = 'item-title';
        titleSpan.textContent = src.name || '未命名源';

        headerDiv.appendChild(toggle);
        headerDiv.appendChild(titleSpan);

        // 底部元信息行：Badge + Category
        const metaDiv = document.createElement('div');
        metaDiv.className = 'item-meta';

        const typeBadge = document.createElement('span');
        typeBadge.className = `badge ${src.type === 'json_api' ? 'json' : 'rss'}`;
        typeBadge.textContent = src.type === 'json_api' ? 'JSON' : 'RSS';
        metaDiv.appendChild(typeBadge);

        if (src.category) {
            const catSpan = document.createElement('span');
            catSpan.className = 'category-tag';
            catSpan.textContent = src.category;
            metaDiv.appendChild(catSpan);
        }

        li.appendChild(headerDiv);
        li.appendChild(metaDiv);

        li.onclick = () => selectSource(idx);
        DOM.sourceList.appendChild(li);
    });

    if (matchedCount === 0 && sources.length > 0) {
        const emptyLi = document.createElement('li');
        emptyLi.className = 'source-list-empty';
        emptyLi.textContent = '无匹配的新闻源';
        DOM.sourceList.appendChild(emptyLi);
    }
}

function selectSource(index) {
    if (activeSourceIndex !== -1 && activeSourceIndex !== index) {
        syncFormToState();
    }

    activeSourceIndex = index;
    renderSourceList();

    const src = currentConfig.sources[index];
    if (!src) return;

    DOM.emptyState.style.display = 'none';
    DOM.form.style.display = 'block';
    DOM.btnFetchTest.style.display = 'inline-flex';

    // 填充表单字段
    DOM.fName.value = src.name || '';
    DOM.fCategory.value = src.category || '';
    DOM.fUrl.value = src.url || '';
    DOM.fEnabled.checked = src.enabled !== false;
    DOM.fType.value = src.type || 'rss';

    // 填充 Headers
    DOM.headersContainer.innerHTML = '';
    if (src.headers) {
        for (const [k, v] of Object.entries(src.headers)) {
            addHeaderRow(k, v);
        }
    }

    // JSON API 专属配置
    if (src.type === 'json_api') {
        DOM.jsonSection.style.display = 'block';
        const jc = src.json_config || {};
        DOM.fjItemsPath.value = jc.items_path || '';
        DOM.fjTitle.value = jc.title_field || '';
        DOM.fjSummary.value = jc.summary_field || '';
        DOM.fjTime.value = jc.time_field || '';
        DOM.fjLink.value = jc.link_field || '';
        DOM.fjLinkTpl.value = jc.link_template || '';
    } else {
        DOM.jsonSection.style.display = 'none';
    }
}

function addHeaderRow(k, v) {
    const row = document.createElement('div');
    row.className = 'form-group row header-item-row';
    row.innerHTML = `
        <input type="text" class="header-key" placeholder="Key (例如: User-Agent)" value="${k}" style="flex:1;">
        <input type="text" class="header-val" placeholder="Value" value="${v}" style="flex:2;">
        <button type="button" class="btn btn-outline btn-del-header" style="color:var(--accent-error); border-color:var(--accent-error)">✕</button>
    `;
    row.querySelector('.btn-del-header').onclick = () => { row.remove(); syncFormToState(); };
    DOM.headersContainer.appendChild(row);
}

function syncFormToState() {
    if (activeSourceIndex < 0 || activeSourceIndex >= currentConfig.sources.length) return;

    const src = currentConfig.sources[activeSourceIndex];
    src.name = DOM.fName.value;
    src.category = DOM.fCategory.value;
    src.url = DOM.fUrl.value;
    src.enabled = DOM.fEnabled.checked;

    // Headers
    const hKeys = DOM.headersContainer.querySelectorAll('.header-key');
    const hVals = DOM.headersContainer.querySelectorAll('.header-val');
    src.headers = {};
    for (let i = 0; i < hKeys.length; i++) {
        const k = hKeys[i].value.trim();
        const v = hVals[i].value.trim();
        if (k) src.headers[k] = v;
    }
    if (Object.keys(src.headers).length === 0) delete src.headers;

    if (src.type === 'json_api') {
        src.json_config = {
            ...(src.json_config || {}),
            items_path: DOM.fjItemsPath.value,
            title_field: DOM.fjTitle.value,
            summary_field: DOM.fjSummary.value,
            time_field: DOM.fjTime.value,
            link_field: DOM.fjLink.value,
            link_template: DOM.fjLinkTpl.value
        };
        for (const key in src.json_config) {
            if (!src.json_config[key]) delete src.json_config[key];
        }
    }
}

function createNewSource(type) {
    syncFormToState();
    const newSrc = {
        name: type === 'rss' ? '新建 RSS 订阅' : '新建 JSON 接口',
        type: type,
        url: '',
        enabled: true,
        category: '行业动态'
    };
    if (type === 'json_api') newSrc.json_config = {};

    currentConfig.sources.unshift(newSrc);
    searchQuery = '';
    DOM.searchSources.value = '';
    selectSource(0);
}

function setupRealtimeSync() {
    const inputs = DOM.form.querySelectorAll('input');
    inputs.forEach(input => {
        input.addEventListener('input', () => {
            syncFormToState();
            renderSourceList();
        });
    });
}

function deleteCurrentSource() {
    if (activeSourceIndex < 0) return;
    const src = currentConfig.sources[activeSourceIndex];
    if (confirm(`确定要删除新闻源「${src.name || '未命名'}」吗？`)) {
        currentConfig.sources.splice(activeSourceIndex, 1);
        activeSourceIndex = -1;
        DOM.emptyState.style.display = 'flex';
        DOM.form.style.display = 'none';
        DOM.btnFetchTest.style.display = 'none';
        renderSourceList();
        showStatus('🗑 源已移除（记得点击保存）');
    }
}

async function saveConfig() {
    syncFormToState();
    DOM.btnSave.disabled = true;
    DOM.btnSave.textContent = "保存中...";
    try {
        const jsonStr = JSON.stringify(currentConfig, null, 2);
        await window.go.main.App.SaveConfig(jsonStr);
        showStatus('✔️ 配置已保存');
    } catch (e) {
        showStatus('❌ 保存失败: ' + e, true);
    } finally {
        DOM.btnSave.disabled = false;
        DOM.btnSave.textContent = "💾 保存配置";
    }
}

async function importOPML() {
    try {
        const addedCount = await window.go.main.App.SelectAndImportOPML();
        if (addedCount > 0) {
            await reloadConfig();
            showStatus(`🎉 成功导入 ${addedCount} 个新订阅源！`);
        } else if (addedCount === 0) {
            showStatus('ℹ️ 未导入新源（已存在或用户取消）');
        }
    } catch (e) {
        showStatus('❌ 导入 OPML 失败: ' + e, true);
    }
}

async function exportOPML() {
    try {
        const opmlStr = await window.go.main.App.ExportOPML();
        if (!opmlStr) return;

        const blob = new Blob([opmlStr], { type: 'application/xml;charset=utf-8;' });
        const link = document.createElement('a');
        link.href = URL.createObjectURL(blob);
        link.download = `newspocket_feeds_${new Date().toISOString().slice(0, 10)}.opml`;
        link.click();
        URL.revokeObjectURL(link.href);
        showStatus('✔️ OPML 导出成功');
    } catch (e) {
        showStatus('❌ 导出失败: ' + e, true);
    }
}

async function openEmailPreview() {
    syncFormToState();
    DOM.previewModal.style.display = 'flex';
    DOM.previewLoading.style.display = 'flex';
    DOM.previewFrame.style.opacity = '0.3';

    try {
        const html = await window.go.main.App.PreviewEmail();
        DOM.previewFrame.srcdoc = html;
        DOM.previewFrame.style.opacity = '1';
    } catch (e) {
        DOM.previewFrame.srcdoc = `<div style="color:#ef4444;padding:30px;font-family:sans-serif;"><h3>❌ 渲染预览失败</h3><p>${e}</p></div>`;
        DOM.previewFrame.style.opacity = '1';
    } finally {
        DOM.previewLoading.style.display = 'none';
    }
}

async function testFetch() {
    syncFormToState();
    const src = currentConfig.sources[activeSourceIndex];

    DOM.btnFetchTest.disabled = true;
    DOM.btnFetchTest.textContent = "抓取中...";

    try {
        const res = await window.go.main.App.TestSource(JSON.stringify(src));
        DOM.testResultText.textContent = res;
        DOM.testModal.style.display = 'flex';
    } catch (e) {
        DOM.testResultText.textContent = "❌ " + e;
        DOM.testModal.style.display = 'flex';
    } finally {
        DOM.btnFetchTest.disabled = false;
        DOM.btnFetchTest.textContent = "▶️ 测试抓取当前源";
    }
}

function showStatus(msg, isError = false) {
    DOM.saveStatus.textContent = msg;
    DOM.saveStatus.style.color = isError ? 'var(--accent-error)' : 'var(--accent-info)';
    DOM.saveStatus.style.opacity = 1;
    setTimeout(() => { DOM.saveStatus.style.opacity = 0; }, 3500);
}

async function openLocalConfig() {
    try {
        const configStr = await window.go.main.App.SelectConfigFile();
        if (configStr) {
            currentConfig = JSON.parse(configStr);
            if (!currentConfig.sources) currentConfig.sources = [];
            activeSourceIndex = -1;
            DOM.emptyState.style.display = 'flex';
            DOM.form.style.display = 'none';
            DOM.btnFetchTest.style.display = 'none';
            renderSourceList();
            showStatus('✔️ 已加载新配置文件');
        }
    } catch (e) {
        showStatus('❌ 打开失败: ' + e, true);
    }
}

document.getElementById('btn-delete-source-new').onclick = deleteCurrentSource;

window.addEventListener('load', () => { setTimeout(init, 300); });
