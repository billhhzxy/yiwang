const apiBase = "/api";

const readyList = document.getElementById("ready-list");
const allList = document.getElementById("all-list");
const readyPagination = document.getElementById("ready-pagination");
const allPagination = document.getElementById("all-pagination");
const createForm = document.getElementById("create-form");

// Configure markdown to be GitHub-like and keep line breaks.
if (window.marked) {
  marked.setOptions({
    gfm: true,
    breaks: true,
  });
}

const pageSize = 10;
let readyData = [];
let allData = [];
let readyPage = 1;
let allPage = 1;
function renderReadyPage() {
  renderList(
    readyList,
    readyPagination,
    readyData,
    true,
    readyPage,
    (p) => {
      readyPage = p;
      renderReadyPage();
    }
  );
}
function renderAllPage() {
  renderList(
    allList,
    allPagination,
    allData,
    false,
    allPage,
    (p) => {
      allPage = p;
      renderAllPage();
    }
  );
}

document.getElementById("refresh-ready").addEventListener("click", loadReady);
document.getElementById("refresh-all").addEventListener("click", loadAll);

createForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  const question = document.getElementById("question").value.trim();
  const answer = document.getElementById("answer").value.trim();
  if (!question || !answer) return;
  await api("/tasks", {
    method: "POST",
    body: JSON.stringify({ question, answer }),
  });
  createForm.reset();
  await Promise.all([loadReady(), loadAll()]);
});

async function loadReady() {
  readyData = await api("/tasks/ready");
  readyPage = 1;
  renderReadyPage();
}

async function loadAll() {
  allData = await api("/tasks?status=all");
  allPage = 1;
  renderAllPage();
}

function renderList(container, pager, tasks, showActions, page, onPageChange) {
  container.innerHTML = "";
  pager.innerHTML = "";
  if (!tasks || tasks.length === 0) {
    container.innerHTML = `<div class="empty">暂无</div>`;
    return;
  }
  const pageCount = Math.max(1, Math.ceil(tasks.length / pageSize));
  const currentPage = Math.min(Math.max(1, page), pageCount);
  const start = (currentPage - 1) * pageSize;
  const slice = tasks.slice(start, start + pageSize);

  slice.forEach((t) => {
    const card = document.createElement("div");
    card.className = "task";

    const head = document.createElement("div");
    head.className = "task-head";
    const title = document.createElement("div");
    title.className = "task-title markdown";
    title.innerHTML = renderMarkdown(t.question);
    const meta = document.createElement("div");
    meta.className = "task-meta";
    meta.textContent = `阶段 ${t.stage + 1} / ${t.totalStages} · 状态 ${t.status}`;
    head.append(title, meta);

    const body = document.createElement("div");
    body.className = "task-body";

    const answerSection = document.createElement("div");
    answerSection.className = "task-section answer hidden";
    const answerLabel = document.createElement("strong");
    answerLabel.textContent = "答案";
    const answerContent = document.createElement("div");
    answerContent.className = "markdown";
    answerContent.innerHTML = renderMarkdown(t.answer);
    answerSection.append(answerLabel, answerContent);

    const metaSection = document.createElement("div");
    metaSection.className = "task-section meta";
    metaSection.innerHTML = `<span>下次：${formatTime(t.nextReviewAt)}</span><span>创建：${formatTime(t.createdAt)}</span>`;

    body.append(answerSection, metaSection);

    const toggleBtn = document.createElement("button");
    toggleBtn.className = "btn btn-small btn-ghost";
    toggleBtn.textContent = "显示答案";
    toggleBtn.onclick = () => {
      const hidden = answerSection.classList.toggle("hidden");
      toggleBtn.textContent = hidden ? "显示答案" : "隐藏答案";
    };

    const actions = document.createElement("div");
    actions.className = "task-actions";

    if (showActions) {
      const rememberBtn = document.createElement("button");
      rememberBtn.className = "btn btn-small";
      rememberBtn.textContent = "记住了";
      rememberBtn.onclick = () => review(t.id, "remembered");
      const forgetBtn = document.createElement("button");
      forgetBtn.className = "btn btn-small btn-ghost";
      forgetBtn.textContent = "忘记了";
      forgetBtn.onclick = () => review(t.id, "forgot");
      actions.append(toggleBtn, rememberBtn, forgetBtn);
    } else {
      const deleteBtn = document.createElement("button");
      deleteBtn.className = "btn btn-small btn-danger";
      deleteBtn.textContent = "删除";
      deleteBtn.onclick = () => removeTask(t.id);
      actions.append(toggleBtn, deleteBtn);
    }

    card.append(head, body, actions);
    container.append(card);
  });

  if (pageCount > 1) {
    const pag = document.createElement("div");
    pag.className = "pagination-inner";

    const prev = document.createElement("button");
    prev.className = "page-btn";
    prev.textContent = "上一页";
    prev.disabled = currentPage === 1;
    prev.onclick = () => onPageChange(currentPage - 1);

    const info = document.createElement("span");
    info.className = "page-info";
    info.textContent = `${currentPage} / ${pageCount}`;

    const next = document.createElement("button");
    next.className = "page-btn";
    next.textContent = "下一页";
    next.disabled = currentPage === pageCount;
    next.onclick = () => onPageChange(currentPage + 1);

    pag.append(prev, info, next);
    pager.append(pag);
  }
}

async function review(id, result) {
  await api(`/tasks/${id}/review`, {
    method: "POST",
    body: JSON.stringify({ result }),
  });
  await Promise.all([loadReady(), loadAll()]);
}

async function removeTask(id) {
  if (!confirm("确认删除该任务？")) return;
  await api(`/tasks/${id}`, { method: "DELETE" });
  await Promise.all([loadReady(), loadAll()]);
}

async function api(path, options = {}) {
  const res = await fetch(apiBase + path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) {
    let msg = `请求失败 (${res.status})`;
    try {
      const data = await res.json();
      if (data && data.error) msg = data.error;
    } catch (_) {}
    alert(msg);
    throw new Error(msg);
  }
  if (res.status === 204) return;
  return res.json();
}

function formatTime(t) {
  if (!t) return "无";
  const dt = new Date(t);
  if (Number.isNaN(dt.getTime())) return "无";
  return dt.toLocaleString();
}

function renderMarkdown(md) {
  if (!md) return "";
  return DOMPurify.sanitize(marked.parse(md, { gfm: true, breaks: true }));
}

// 初始加载
loadReady();
loadAll();

