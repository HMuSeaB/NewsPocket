let currentConfig = { sources: [] };
let activeSourceIndex = -1;

const DOM = {
    sourceList: document.getElementById('source-list'),
    form: document.getElementById('source-form'),
    emptyState: document.getElementById('empty-state'),
    jsonSection: document.getElementById('json-config-section'),
    headersContainer: document.getElementById('headers-container'),
    saveStatus: document.getElementById('save-status'),
    btnOpenConfig: document.getElementById('btn-open-config'),
    btnSave: document.getElementById('btn-save'),
    btnAddRss: document.getElementById('btn-add-rss'),
    btnAddJson: document.getElementById('btn-add-json'),
    btnAddHeader: document.getElementById('btn-add-header'),
    btnFetchTest: document.getElementById('btn-fetch-test'),

    // Form fields
    fName: document.getElementById('name'),
    fCategory: document.getElementById('category'),
    fUrl: document.getElementById('url'),
    fEnabled: document.getElementById('enabled'),
    fType: document.getElementById('type'),
    fjItemsPath: document.getElementById('items_path'),
    fjTitle: document.getElementById('title_field'),
    fjTime: document.getElementById('time_field'),
    fjLink: document.getElementById('link_field'),
    fjLinkTpl: document.getElementById('link_template')
};

// Initialize app
async function init() {
    // 强行优先绑定所有事件，无视初始配置是否加载成功
    DOM.btnAddRss.onclick = () => createNewSource('rss');
    DOM.btnAddJson.onclick = () => createNewSource('json_api');
    DOM.btnSave.onclick = saveConfig;
    DOM.btnOpenConfig.onclick = openLocalConfig;
    DOM.btnFetchTest.onclick = testFetch;
    DOM.btnAddHeader.onclick = () => addHeaderRow('', '');
    setup实时保存();

    try {
        const configStr = await window.go.main.App.GetConfig();
        currentConfig = JSON.parse(configStr);
        if (!currentConfig.sources) currentConfig.sources = [];
        renderSourceList();
    } catch (e) {
        showStatus('加载配置失败: 请手动「打开本地配置」', true);
    }
}

function renderSourceList() {
    DOM.sourceList.innerHTML = '';
    currentConfig.sources.forEach((src, idx) => {
        const li = document.createElement('li');
        li.className = `source-item ${idx === activeSourceIndex ? 'active' : ''}`;

        // title
        const titleDiv = document.createElement('div');
        titleDiv.className = 'item-title';
        titleDiv.textContent = src.name || '未命名源';
        if (src.enabled === false) {
            titleDiv.style.textDecoration = 'line-through';
            titleDiv.style.opacity = '0.5';
        }

        // meta
        const metaDiv = document.createElement('div');
        metaDiv.className = 'item-meta';
        const typeBadge = document.createElement('span');
        typeBadge.className = `badge ${src.type === 'json_api' ? 'json' : 'rss'}`;
        typeBadge.textContent = src.type;
        metaDiv.appendChild(typeBadge);

        if (src.category) {
            const catSpan = document.createElement('span');
            catSpan.textContent = src.category;
            metaDiv.appendChild(catSpan);
        }

        li.appendChild(titleDiv);
        li.appendChild(metaDiv);

        li.onclick = () => selectSource(idx);
        DOM.sourceList.appendChild(li);
    });
}

function selectSource(index) {
    if (activeSourceIndex !== -1 && activeSourceIndex !== index) {
        syncFormToState(); // save previous
    }

    activeSourceIndex = index;
    renderSourceList();

    const src = currentConfig.sources[index];
    if (!src) return;

    DOM.emptyState.style.display = 'none';
    DOM.form.style.display = 'block';
    DOM.btnFetchTest.style.display = 'inline-block';

    // Populate form
    DOM.fName.value = src.name || '';
    DOM.fCategory.value = src.category || '';
    DOM.fUrl.value = src.url || '';
    DOM.fEnabled.checked = src.enabled !== false; // default true
    DOM.fType.value = src.type || 'rss';

    // Header logic
    DOM.headersContainer.innerHTML = '';
    if (src.headers) {
        for (const [k, v] of Object.entries(src.headers)) {
            addHeaderRow(k, v);
        }
    }

    if (src.type === 'json_api') {
        DOM.jsonSection.style.display = 'block';
        const jc = src.json_config || {};
        DOM.fjItemsPath.value = jc.items_path || '';
        DOM.fjTitle.value = jc.title_field || '';
        DOM.fjTime.value = jc.time_field || '';
        DOM.fjLink.value = jc.link_field || '';
        DOM.fjLinkTpl.value = jc.link_template || '';
    } else {
        DOM.jsonSection.style.display = 'none';
    }
}

function addHeaderRow(k, v) {
    const row = document.createElement('div');
    row.className = 'form-group row';
    row.style.marginBottom = '10px';
    row.innerHTML = `
        <input type="text" class="header-key" placeholder="Key (e.g. User-Agent)" value="${k}" style="flex:1;">
        <input type="text" class="header-val" placeholder="Value" value="${v}" style="flex:2;">
        <button type="button" class="btn btn-outline" onclick="this.parentElement.remove()" style="color:var(--danger-color); border-color:var(--danger-color)">删除</button>
    `;
    DOM.headersContainer.appendChild(row);
}

function syncFormToState() {
    if (activeSourceIndex < 0 || activeSourceIndex >= currentConfig.sources.length) return;

    const src = currentConfig.sources[activeSourceIndex];
    src.name = DOM.fName.value;
    src.category = DOM.fCategory.value;
    src.url = DOM.fUrl.value;
    src.enabled = DOM.fEnabled.checked;

    // Sync headers
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
            items_path: DOM.fjItemsPath.value,
            title_field: DOM.fjTitle.value,
            time_field: DOM.fjTime.value,
            link_field: DOM.fjLink.value,
            link_template: DOM.fjLinkTpl.value
        };
        // Clean empty values
        for (const key in src.json_config) {
            if (!src.json_config[key]) delete src.json_config[key];
        }
    }
}

function createNewSource(type) {
    syncFormToState();
    const newSrc = {
        name: type === 'rss' ? '新建 RSS' : '新建 JSON',
        type: type,
        url: '',
        enabled: true,
        category: ''
    };
    if (type === 'json_api') newSrc.json_config = {};

    currentConfig.sources.push(newSrc);
    selectSource(currentConfig.sources.length - 1);
}

// 绑定所有的输入框事件以实现"编辑后立即同步并在左侧预览"的效果
function setup实时保存() {
    const inputs = DOM.form.querySelectorAll('input');
    inputs.forEach(input => {
        input.addEventListener('input', () => {
            syncFormToState();
            renderSourceList(); // 只重新渲染列表标题
        });
    });
}

function deleteCurrentSource() {
    // Adding Delete Button dynamically in the form header row
    if (activeSourceIndex < 0) return;
    if (confirm("确定要删除这个新闻源吗？不可恢复。")) {
        currentConfig.sources.splice(activeSourceIndex, 1);
        activeSourceIndex = -1;
        DOM.emptyState.style.display = 'flex';
        DOM.form.style.display = 'none';
        DOM.btnFetchTest.style.display = 'none';
        renderSourceList();
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

async function testFetch() {
    syncFormToState();
    const src = currentConfig.sources[activeSourceIndex];

    DOM.btnFetchTest.disabled = true;
    DOM.btnFetchTest.textContent = "抓取中...";

    try {
        const res = await window.go.main.App.TestSource(JSON.stringify(src));
        document.getElementById('test-result-text').textContent = res;
        document.getElementById('test-result-modal').style.display = 'flex';
    } catch (e) {
        document.getElementById('test-result-text').textContent = "❌ " + e;
        document.getElementById('test-result-modal').style.display = 'flex';
    } finally {
        DOM.btnFetchTest.disabled = false;
        DOM.btnFetchTest.textContent = "▶️ 测试抓取";
    }
}

function showStatus(msg, isError = false) {
    DOM.saveStatus.textContent = msg;
    DOM.saveStatus.style.color = isError ? 'var(--danger-color)' : 'var(--accent-green)';
    DOM.saveStatus.style.opacity = 1;
    setTimeout(() => { DOM.saveStatus.style.opacity = 0; }, 3000);
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
            showStatus('✔️ 已加载新配置');
        }
    } catch (e) {
        showStatus('❌ 打开失败: ' + e, true);
    }
}

// Add delete button event
document.getElementById('btn-delete-source-new').onclick = deleteCurrentSource;


// Wails injects runtime, so we wait for it
window.addEventListener('load', () => { setTimeout(init, 500); });
