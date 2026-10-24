"use strict";
// pm's interface. Every text comes from pm's catalogs (/api/i18n) through t().

// ---- basics -------------------------------------------------------------------

let M = {};
let LANG = "fr";
function t(key, ...args) {
  let s = M[key] || key;
  for (const a of args) s = s.replace(/%[sdvq]/, String(a));
  return s;
}

async function api(method, path, body) {
  const opts = { method, headers: { "X-PM": "1" } };
  if (body !== undefined) {
    opts.headers["Content-Type"] = "application/json";
    opts.body = JSON.stringify(body);
  }
  const resp = await fetch(path, opts);
  const data = await resp.json().catch(() => ({}));
  if (!resp.ok) throw new Error(data.error || resp.statusText);
  return data;
}
const get = (path) => api("GET", path);
const post = (path, body) => api("POST", path, body || {});
const q = (o) => new URLSearchParams(o).toString();

// el builds an element: el("button", {class, onclick, tip}, "text", child...).
function el(tag, attrs, ...children) {
  const e = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (v === undefined || v === null || v === false) continue;
    if (k.startsWith("on")) e.addEventListener(k.slice(2), v);
    else if (k === "class") e.className = v;
    else if (k === "tip") e.dataset.tip = v;
    else if (v === true) e.setAttribute(k, "");
    else e.setAttribute(k, v);
  }
  add(e, children);
  return e;
}
function add(parent, children) {
  for (const c of children.flat(Infinity)) {
    if (c === undefined || c === null || c === false || c === "") continue;
    parent.append(c instanceof Node ? c : document.createTextNode(String(c)));
  }
  return parent;
}

// Icons drawn for pm (24×24, stroked), no icon font to license.
const ICONS = {
  play: '<path d="M7 5l12 7-12 7z"/>',
  pause: '<path d="M8 5v14M16 5v14"/>',
  stop: '<rect x="6" y="6" width="12" height="12" rx="1"/>',
  restart: '<path d="M20 12a8 8 0 1 1-2.3-5.7"/><path d="M20 4v5h-5"/>',
  detach: '<path d="M15 7h2a5 5 0 0 1 0 10h-2M9 17H7A5 5 0 0 1 7 7h2"/><path d="M8 12h2M14 12h2"/>',
  log: '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M7 9l3 3-3 3M13 15h4"/>',
  sliders: '<path d="M4 6h9M17 6h3M4 12h3M11 12h9M4 18h11M19 18h1"/><circle cx="15" cy="6" r="2"/><circle cx="9" cy="12" r="2"/><circle cx="17" cy="18" r="2"/>',
  sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M2 12h2M20 12h2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>',
  moon: '<path d="M20 14.5A8 8 0 0 1 9.5 4a8 8 0 1 0 10.5 10.5z"/>',
  book: '<path d="M4 19V5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2z"/><path d="M8 7h7"/>',
  keyboard: '<rect x="2" y="6" width="20" height="12" rx="2"/><path d="M6 10h.01M10 10h.01M14 10h.01M18 10h.01M7 14h10"/>',
  folder: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  x: '<path d="M6 6l12 12M18 6L6 18"/>',
  check: '<path d="M5 12l5 5 9-10"/>',
  alert: '<path d="M12 3l10 18H2z"/><path d="M12 10v5M12 18v.01"/>',
  eye: '<path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/>',
  link: '<path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1"/><path d="M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1"/>',
  pencil: '<path d="M4 20h4L19 9l-4-4L4 16z"/><path d="M13 7l4 4"/>',
  trash: '<path d="M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13"/>',
  search: '<circle cx="11" cy="11" r="6"/><path d="M16 16l4 4"/>',
  left: '<path d="M15 6l-6 6 6 6"/>',
  right: '<path d="M9 6l6 6-6 6"/>',
  wand: '<path d="M4 20L15 9"/><path d="M15 4v2M19 8h2M18 5l1.5-1.5M12 6l-.5-1M17 11l1 .5"/>',
  gamepad: '<path d="M6 8h12a3 3 0 0 1 3 3l-1 5.5a2 2 0 0 1-3.5 1L14 15h-4l-2.5 2.5a2 2 0 0 1-3.5-1L3 11a3 3 0 0 1 3-3z"/><path d="M8 10v3M6.5 11.5h3M16 11h.01M18 13h.01"/>',
  pulse: '<path d="M3 12h4l2-5 4 10 2-5h6"/>',
  archive: '<rect x="3" y="4" width="18" height="4" rx="1"/><path d="M5 8v11h14V8M10 12h4"/>',
  copy: '<rect x="8" y="8" width="12" height="12" rx="2"/><path d="M16 8V5a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h3"/>',
  info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v5M12 8v.01"/>',
  file: '<path d="M6 3h8l4 4v14H6z"/><path d="M14 3v4h4"/>',
  user: '<circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/>',
  terminal: '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M7 9l3 3-3 3"/>',
};
function icon(name) {
  const s = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  s.setAttribute("viewBox", "0 0 24 24");
  s.setAttribute("class", "i");
  s.setAttribute("aria-hidden", "true");
  s.innerHTML = ICONS[name] || "";
  return s;
}
function btn(label, attrs, iconName) {
  const cls = "btn " + ((attrs && attrs.class) || "");
  return el("button", { type: "button", ...attrs, class: cls.trim() }, iconName && icon(iconName), label);
}
function iconBtn(iconName, tip, onclick) {
  return el("button", { type: "button", class: "btn icon", tip, "aria-label": tip, onclick }, icon(iconName));
}
// browse opens the Windows file window and fills input with the path
// chosen: pm opens it, as a page is never told where a file lives. kind is
// what the field holds (program, file, folder, icon, yaml, newyaml, save),
// or a function giving it.
function browse(input, kind, then, labeled) {
  const run = () => busy(b, async () => {
    try {
      const typed = input.value.trim() || (/^[a-z]:\\/i.test(input.placeholder) ? input.placeholder : "");
      const r = await post("/api/pick", { kind: typeof kind === "function" ? kind() : kind, start: typed });
      if (!r.path) return;
      input.value = r.path;
      input.dispatchEvent(new Event("input", { bubbles: true }));
      input.dispatchEvent(new Event("change", { bubbles: true }));
      then ? then(r.path) : input.focus();
    } catch (e) { failToast(e); }
  });
  const b = labeled ? btn(t("gui.browse"), { onclick: run }, "folder") : iconBtn("folder", t("gui.browse"), run);
  return b;
}
function withBrowse(input, kind) { return el("div", { class: "row-line" }, el("div", { class: "grow" }, input), browse(input, kind, null, true)); }
// busy shows a button working while fn runs.
async function busy(button, fn) {
  const before = [...button.childNodes];
  button.disabled = true;
  button.replaceChildren(el("span", { class: "spinner" }), ...before.filter((n) => n.nodeType === 3));
  try { return await fn(); } finally { button.disabled = false; button.replaceChildren(...before); }
}
function failBanner(e) { return el("div", { class: "banner bad" }, icon("alert"), el("span", { class: "grow" }, e.message || String(e))); }

function fmtTime(iso) { return new Date(iso).toLocaleTimeString(LANG); }
function fmtDate(iso) { return new Date(iso).toLocaleString(LANG); }
function fmtSize(n) { return n < 1024 ? t("gui.bytes", n) : t("gui.kb", Math.round(n / 1024)); }
function base(p) { return p.split("\\").pop(); }
function folderOf(p) { return p.split("\\").slice(0, -1).join("\\"); }

// ---- toasts and dialogs -----------------------------------------------------

function toast(text, kind) {
  const box = document.getElementById("toasts");
  const n = el("div", { class: "toast " + (kind || ""), role: "status" }, icon(kind === "bad" ? "alert" : "check"), el("span", {}, text));
  box.append(n);
  setTimeout(() => n.remove(), kind === "bad" ? 9000 : 4500);
}
function failToast(e) { toast(e.message || String(e), "bad"); }

const dialogs = [];
function dialog(title, ...content) {
  const back = el("div", { class: "dialog-back", onmousedown: (e) => { if (e.target === back) entry.close(); } });
  const close = () => { back.remove(); const i = dialogs.indexOf(entry); if (i >= 0) dialogs.splice(i, 1); };
  const entry = { close };
  back.append(el("div", { class: "dialog", role: "dialog", "aria-modal": "true", "aria-label": title },
    el("header", {}, el("h2", {}, title), iconBtn("x", t("gui.close"), () => entry.close())),
    add(el("div", { class: "body" }), content)));
  document.body.append(back);
  dialogs.push(entry);
  const first = back.querySelector(".body input, .body select, .body textarea, .body .btn.primary");
  if (first) first.focus();
  return entry;
}
// confirmBox asks a yes/no question in the page; resolves true for yes.
function confirmBox(title, text, yesLabel, danger) {
  return new Promise((resolve) => {
    let entry;
    // The answer first: closing resolves false, which a settled promise ignores.
    const answer = (v) => { resolve(v); entry.close(); };
    entry = dialog(title, el("p", { class: "intro" }, text),
      el("div", { class: "btn-row end" },
        btn(t("gui.cancel"), { class: "text", onclick: () => answer(false) }),
        btn(yesLabel, { class: danger ? "primary danger" : "primary", onclick: () => answer(true) })));
    const close = entry.close;
    entry.close = () => { close(); resolve(false); };
  });
}
function closeAll() { while (dialogs.length) dialogs[dialogs.length - 1].close(); }

// field is a label, its technical name, the input, and a line of help.
function field(label, key, hint, input) {
  return el("label", { class: "field" },
    el("span", { class: "label-line" }, el("span", { class: "eyebrow" }, label), key && el("span", { class: "key" }, key)),
    input, hint && el("span", { class: "hint" }, hint));
}
function main(...content) { document.getElementById("main").replaceChildren(...content); }

// ---- shell: tabs, tools, footer, keys -----------------------------------------

const S = { tab: "sessions", config: null, sub: "overview", profile: null, mode: null, dirty: false };
let live = []; // timers and streams of the current view
function stopLive() { for (const x of live) (x.close ? x.close() : clearInterval(x)); live = []; }

async function leaveEdits() {
  if (!S.dirty) return true;
  if (!(await confirmBox(t("gui.unsaved_title"), t("gui.unsaved_text"), t("gui.discard"), true))) return false;
  S.dirty = false;
  return true;
}

async function show(tab) {
  if (tab !== S.tab && !(await leaveEdits())) return;
  stopLive();
  for (const name of Object.keys(logs)) closeLog(name);
  S.tab = tab;
  history.replaceState(null, "", "#" + tab);
  for (const b of document.querySelectorAll("#tabs button")) b.classList.toggle("on", b.dataset.tab === tab);
  footer();
  ({ sessions: viewSessions, configs: viewConfigs, wizard: viewWizard })[tab]();
}

function footer() {
  const k = (keys, label) => el("span", {}, keys.map((x) => el("kbd", {}, x)), label);
  const items = [k(["1", "2", "3"], t("gui.key_tabs"))];
  if (S.tab === "configs") items.push(k(["/"], t("gui.key_filter")));
  if (S.tab === "wizard") items.push(k([t("gui.key_enter")], t("gui.key_next")));
  items.push(k([t("gui.key_esc")], t("gui.key_close")));
  document.getElementById("foot").replaceChildren(...items);
}

function tools() {
  const light = () => document.documentElement.classList.contains("light");
  const theme = iconBtn(light() ? "moon" : "sun", t("gui.theme"), () => {
    const on = !light();
    document.documentElement.classList.toggle("light", on);
    try { localStorage.setItem("pm-theme", on ? "light" : "dark"); } catch {}
    theme.replaceChildren(icon(on ? "moon" : "sun"));
  });
  const doc = el("a", { class: "btn icon", tip: t("gui.doc"), "aria-label": t("gui.doc"), target: "_blank", rel: "noopener",
    href: LANG === "en" ? "https://davidp57.github.io/process-monitor/latest/en/gui/" : "https://davidp57.github.io/process-monitor/latest/gui/" }, icon("book"));
  document.getElementById("tools").replaceChildren(
    theme, iconBtn("sliders", t("gui.settings"), () => settings()), iconBtn("keyboard", t("gui.shortcuts_help"), help), doc);
}

function help() {
  const row = (keys, text) => el("div", { class: "row-line" }, el("span", { style: "min-width:110px" }, keys.map((x) => el("kbd", {}, x))), el("span", {}, text));
  dialog(t("gui.shortcuts_help"), el("div", { class: "rows" },
    row(["1", "2", "3"], t("gui.key_tabs_long")), row(["/"], t("gui.key_filter_long")),
    row([t("gui.key_enter")], t("gui.key_next_long")), row([t("gui.key_esc")], t("gui.key_close_long"))));
}

document.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && dialogs.length) { dialogs[dialogs.length - 1].close(); e.preventDefault(); return; }
  const typing = /INPUT|TEXTAREA|SELECT/.test(e.target.tagName);
  if (typing || dialogs.length || e.ctrlKey || e.altKey || e.metaKey) return;
  const tabs = { 1: "sessions", 2: "configs", 3: "wizard" };
  if (tabs[e.key]) { show(tabs[e.key]); e.preventDefault(); }
  if (e.key === "/" && S.tab === "configs") { document.getElementById("filter")?.focus(); e.preventDefault(); }
});

// ---- settings -----------------------------------------------------------------

async function settings(tab) {
  let data, i;
  try { [data, i] = await Promise.all([get("/api/configs"), get("/api/i18n")]); } catch (e) { failToast(e); return; }
  const body = el("div");
  const sub = el("div", { class: "subtabs" });
  let which = tab || "general";
  const render = () => {
    sub.replaceChildren(...[["general", t("gui.settings_general")], ["folders", t("gui.settings_folders")]].map(([k, label]) =>
      el("button", { type: "button", class: which === k ? "on" : "", onclick: () => { which = k; render(); } }, label)));
    body.replaceChildren(which === "general" ? generalSettings(i) : folderSettings(data));
  };
  render();
  dialog(t("gui.settings"), sub, body, el("p", { class: "hint", style: "margin-top:14px" }, t("gui.settings_file")));
}

function generalSettings(i) {
  const lang = el("select", {},
    el("option", { value: "" }, t("gui.lang_windows")),
    el("option", { value: "fr", selected: i.setting === "fr" }, "Français"),
    el("option", { value: "en", selected: i.setting === "en" }, "English"));
  lang.addEventListener("change", async () => {
    try { await post("/api/lang", { lang: lang.value }); location.reload(); } catch (e) { failToast(e); }
  });
  return el("div", {}, field(t("gui.lang"), null, t("gui.lang_hint"), lang));
}

function folderSettings(data) {
  const input = el("input", { type: "text", class: "mono", placeholder: t("gui.folder_path") });
  const addBtn = btn(t("gui.add_folder"), { onclick: () => busy(addBtn, async () => {
    try { await post("/api/folders/add", { path: input.value }); toast(t("gui.folder_added")); closeAll(); settings("folders"); refreshConfigs(); }
    catch (e) { failToast(e); }
  }) }, "plus");
  input.addEventListener("keydown", (e) => { if (e.key === "Enter") addBtn.click(); });
  return el("div", {},
    el("p", { class: "intro" }, t("gui.folders_intro")),
    el("div", { class: "eyebrow" }, t("gui.folders")),
    el("div", { class: "rows", style: "margin:6px 0 12px" },
      data.folders.length ? data.folders.map((f) => el("div", { class: "row-line" }, icon("folder"), el("span", { class: "grow mono" }, f),
        iconBtn("trash", t("gui.remove"), async () => {
          try { await post("/api/folders/remove", { path: f }); closeAll(); settings("folders"); refreshConfigs(); } catch (e) { failToast(e); }
        }))) : el("span", { class: "hint" }, t("gui.no_folder"))),
    el("div", { class: "row-line" }, el("div", { class: "grow" }, input), browse(input, "folder"), addBtn),
    el("div", { class: "eyebrow", style: "margin-top:18px" }, t("gui.default_folders")),
    el("p", { class: "hint" }, t("gui.default_folders_hint")),
    el("div", { class: "rows" }, data.defaults.map((f) => el("div", { class: "row-line muted" }, icon("folder"), el("span", { class: "mono" }, f)))));
}

// ---- sessions -----------------------------------------------------------------

const logs = {}; // session name → its open log element, kept across refreshes

async function viewSessions() {
  const list = el("div");
  const count = el("span", { class: "pill" });
  main(el("div", { class: "page" },
    el("div", { class: "page-head" }, el("h2", {}, t("gui.sessions")), count),
    el("p", { class: "intro" }, t("gui.sessions_intro")), list));
  async function refresh() {
    let sessions;
    try { sessions = await get("/api/sessions"); } catch (e) { list.replaceChildren(failBanner(e)); return; }
    count.textContent = t("gui.n_running", sessions.length);
    for (const name of Object.keys(logs)) if (!sessions.find((s) => s.name === name)) closeLog(name);
    if (!sessions.length) {
      list.replaceChildren(el("div", { class: "card empty" }, icon("pulse"), el("h3", {}, t("gui.no_session")),
        el("p", {}, t("gui.no_session_hint")), btn(t("gui.go_configs"), { class: "primary", onclick: () => show("configs") }, "folder")));
      return;
    }
    list.replaceChildren(...sessions.map((s) => sessionCard(s, refresh)));
  }
  await refresh();
  live.push(setInterval(refresh, 2000));
}

function statusPill(status) {
  const kind = { running: "ok", paused: "warn", restarting: "warn", starting: "info", stopping: "info" }[status] || "";
  return el("span", { class: "pill " + kind }, el("span", { class: "dot " + (status === "running" ? "live" : "") }), t("status." + status));
}

function sessionCard(s, refresh) {
  const st = s.state, info = s.info, watch = info.mode === "watch";
  const cmd = (c, label) => async (e) => {
    const b = e.currentTarget;
    if (c === "detach" && !(await confirmBox(t("gui.detach"), t("gui.ask_detach"), t("gui.detach"), true))) return;
    try { await busy(b, () => post("/api/sessions/command", { name: s.name, command: c })); toast(t("gui.done_command", label, info.name)); refresh(); }
    catch (err) { failToast(err); }
  };
  const actions = [];
  if (watch && st.Status === "running") actions.push(btn(t("gui.pause"), { onclick: cmd("pause", t("gui.pause")) }, "pause"));
  if (st.Status === "paused") actions.push(btn(t("gui.resume"), { onclick: cmd("resume", t("gui.resume")) }, "play"));
  if (watch && st.Status === "running") actions.push(btn(t("gui.restart"), { onclick: cmd("restart", t("gui.restart")) }, "restart"));
  if (["running", "paused", "restarting"].includes(st.Status)) actions.push(btn(t("gui.stop"), { tip: t("gui.stop_tip"), onclick: cmd("stop", t("gui.stop")) }, "stop"));
  if (["running", "paused"].includes(st.Status)) actions.push(btn(t("gui.detach"), { class: "text", tip: t("gui.detach_tip"), onclick: cmd("detach", t("gui.detach")) }, "detach"));
  const open = !!logs[s.name];
  const toggle = btn(open ? t("gui.hide_log") : t("gui.show_log"), { class: "text small", onclick: () => { open ? closeLog(s.name) : openLog(s.name); refresh(); } }, "log");
  const meta = [];
  if (st.StartedAt && !st.StartedAt.startsWith("0001")) meta.push(el("span", {}, t("gui.since", fmtTime(st.StartedAt))));
  if (st.PID) meta.push(el("span", {}, t("gui.pid", st.PID)));
  if (st.Restarts) meta.push(el("span", {}, t("gui.restarts", st.Restarts)));
  if (st.Ready) meta.push(el("span", {}, t(st.Ready)));
  return el("div", { class: "card" },
    el("div", { class: "card-head" }, el("h3", {}, info.name), info.profile && el("span", { class: "pill info" }, info.profile), statusPill(st.Status),
      el("span", { class: "pill" }, watch ? t("gui.mode_watch_short") : t("gui.mode_run_short")), el("span", { class: "spacer" }), actions),
    el("div", { class: "meta" }, el("span", { class: "mono" }, icon("file"), info.config), meta),
    el("div", { class: "btn-row", style: "margin-top:8px" }, toggle), logs[s.name]);
}
function openLog(name) {
  const box = el("div", { class: "log" });
  logs[name] = box;
  get("/api/sessions/log?" + q({ name })).then((lines) => { for (const l of lines) logLine(box, l); }).catch(failToast);
  const es = new EventSource("/api/sessions/events?" + q({ name }));
  es.onmessage = (ev) => logLine(box, JSON.parse(ev.data));
  box.stop = () => es.close();
}
function closeLog(name) { logs[name]?.stop(); delete logs[name]; }
function logLine(box, l) {
  const stick = box.scrollTop + box.clientHeight >= box.scrollHeight - 20;
  box.append(el("div", { class: (l.Level || "").trim() }, el("span", { class: "t" }, fmtTime(l.Time) + " "), l.Text));
  if (stick) box.scrollTop = box.scrollHeight;
}

// ---- configs --------------------------------------------------------------------

let CONFIGS = [];
function refreshConfigs() { if (S.tab === "configs") viewConfigs(); }
function refreshConfigsLater() { get("/api/configs").then((d) => { CONFIGS = d.configs; }).catch(() => {}); }

async function viewConfigs() {
  let data;
  try { data = await get("/api/configs"); } catch (e) { main(el("div", { class: "page" }, failBanner(e))); return; }
  CONFIGS = data.configs;
  if (!CONFIGS.find((c) => c.path === S.config)) S.config = CONFIGS[0]?.path || null;
  const filter = el("input", { type: "text", id: "filter", placeholder: t("gui.filter"), "aria-label": t("gui.filter") });
  const list = el("div", { class: "list", role: "listbox" });
  const detail = el("section");
  const select = async (p) => {
    if (p === S.config || !(await leaveEdits())) return;
    S.config = p; S.sub = "overview"; S.profile = null; S.mode = null;
    fill();
    detail.replaceChildren(configDetail());
  };
  const fill = () => {
    const f = filter.value.toLowerCase();
    const shown = CONFIGS.filter((c) => !f || (c.name + " " + c.path).toLowerCase().includes(f));
    list.replaceChildren(...shown.map((c) => {
      const ic = c.profiles.find((p) => p.icon)?.icon;
      return el("div", { class: "list-item" + (c.path === S.config ? " on" : ""), role: "option", tabindex: "0", "aria-selected": String(c.path === S.config),
        onclick: () => select(c.path), onkeydown: (e) => { if (e.key === "Enter") select(c.path); } },
        ic ? el("img", { class: "thumb", src: "/api/icon?" + q({ path: ic }), alt: "" }) : el("span", { class: "thumb blank" }, icon("file")),
        el("div", { class: "grow" }, el("div", { class: "title" }, c.name || base(c.path)), el("div", { class: "sub" }, folderOf(c.path))),
        c.error ? el("span", { class: "pill bad", tip: t("gui.invalid") }, "!") : "");
    }));
    if (!shown.length && CONFIGS.length) list.append(el("p", { class: "hint", style: "padding:8px" }, t("gui.no_match")));
  };
  filter.addEventListener("input", fill);
  fill();
  main(el("div", { class: "split" },
    el("aside", {},
      el("div", { class: "aside-head" }, el("span", { class: "eyebrow" }, t("gui.n_configs", CONFIGS.length)),
        el("div", { class: "input-icon" }, icon("search"), filter)),
      list,
      el("div", { class: "aside-foot btn-row" },
        btn(t("gui.add_config"), { class: "text small", onclick: addConfig }, "plus"),
        btn(t("gui.folders_btn"), { class: "text small", onclick: () => settings("folders") }, "folder"))),
    detail));
  detail.append(configDetail());
}

function addConfig() {
  const input = el("input", { type: "text", class: "mono", placeholder: t("gui.config_path") });
  let d;
  const ok = btn(t("gui.add"), { class: "primary", onclick: () => busy(ok, async () => {
    try {
      await post("/api/configs/add", { path: input.value });
      d.close();
      S.config = input.value.trim().replace(/^"|"$/g, "");
      toast(t("gui.config_added"));
      refreshConfigs();
    } catch (e) { failToast(e); }
  }) }, "plus");
  input.addEventListener("keydown", (e) => { if (e.key === "Enter") ok.click(); });
  d = dialog(t("gui.add_config"), field(t("gui.config_file"), null, t("gui.add_config_hint"), withBrowse(input, "yaml")), el("div", { class: "btn-row end" }, ok));
}

function configDetail() {
  const c = CONFIGS.find((x) => x.path === S.config);
  if (!c) {
    return el("div", { class: "detail" }, el("div", { class: "card empty" }, icon("folder"), el("h3", {}, t("gui.no_config")),
      el("p", {}, t("gui.no_config_hint")),
      el("div", { class: "btn-row", style: "justify-content:center" },
        btn(t("gui.tab_wizard"), { class: "primary", onclick: () => show("wizard") }, "wand"),
        btn(t("gui.add_config"), { onclick: addConfig }, "plus"),
        btn(t("gui.folders_btn"), { onclick: () => settings("folders") }, "folder"))));
  }
  if (c.profiles.length && !c.profiles.find((p) => p.name === S.profile)) S.profile = c.profiles[0].name;
  const copy = iconBtn("copy", t("gui.copy_path"), () => { navigator.clipboard?.writeText(c.path); toast(t("gui.copied")); });
  const head = el("div", { class: "detail-head" },
    el("div", { class: "btn-row" }, el("h2", {}, c.name || base(c.path)),
      c.mode && el("span", { class: "pill", tip: t("gui.mode_tip") }, c.mode === "watch" ? t("gui.mode_watch_short") : t("gui.mode_run_short")),
      c.ui && el("span", { class: "pill" }, c.ui === "tray" ? t("gui.ui_tray_short") : t("gui.ui_console_short")),
      !c.remembered && el("span", { class: "pill", tip: t("gui.from_folder_tip") }, t("gui.from_folder")),
      c.error && el("span", { class: "pill bad" }, t("gui.invalid"))),
    el("div", { class: "path-line" }, el("span", { class: "mono" }, c.path), copy));
  const forget = c.remembered && btn(t("gui.forget"), { class: "text small", tip: t("gui.forget_tip"), onclick: async () => {
    if (!(await confirmBox(t("gui.forget"), t("gui.ask_forget", c.path), t("gui.forget"), false))) return;
    try { await post("/api/configs/forget", { path: c.path }); toast(t("gui.forgotten")); refreshConfigs(); } catch (e) { failToast(e); }
  } }, "x");
  if (c.error) {
    return el("div", { class: "detail" }, head, failBanner(new Error(c.error)),
      el("p", { class: "intro" }, t("gui.invalid_hint")),
      el("div", { class: "btn-row" }, S.sub !== "edit" && btn(t("gui.edit"), { class: "primary", onclick: () => { S.sub = "edit"; rerender(); } }, "pencil"), forget),
      S.sub === "edit" ? editor(c) : "");
  }
  const tabs = [["overview", t("gui.sub_overview"), "eye"], ["shortcuts", t("gui.shortcuts"), "link"], ["logs", t("gui.logs"), "log"], ["edit", t("gui.edit"), "pencil"]];
  const sub = el("div", { class: "subtabs", role: "tablist" }, tabs.map(([k, label, ic]) =>
    el("button", { type: "button", role: "tab", "aria-selected": String(S.sub === k), class: S.sub === k ? "on" : "", onclick: async () => {
      if (k === S.sub || !(await leaveEdits())) return;
      S.sub = k; rerender();
    } }, icon(ic), label)), el("span", { class: "spacer" }), forget);
  const body = { overview, shortcuts: shortcutsTab, logs: logsTab, edit: editor }[S.sub](c);
  return el("div", { class: "detail" }, head, launchBlock(c), sub, body);
}
function rerender() {
  const section = document.querySelector(".split > section");
  if (section) section.replaceChildren(configDetail());
}

function profileSeg(c, onchange) {
  if (!c.profiles.length) return null;
  const seg = el("div", { class: "seg", role: "radiogroup", "aria-label": t("gui.profile") });
  const fill = () => seg.replaceChildren(...c.profiles.map((p) => el("button", { type: "button", role: "radio", "aria-checked": String(p.name === S.profile),
    class: p.name === S.profile ? "on" : "", onclick: () => { S.profile = p.name; fill(); onchange && onchange(); } },
    p.icon ? el("img", { class: "thumb", src: "/api/icon?" + q({ path: p.icon }), alt: "" }) : null, p.name)));
  fill();
  return seg;
}

function launchBlock(c) {
  if (!S.mode) S.mode = c.mode || "run";
  const modes = el("div", { class: "seg", role: "radiogroup", "aria-label": t("gui.mode") });
  const fillModes = () => modes.replaceChildren(...[["run", t("gui.mode_run")], ["watch", t("gui.mode_watch")]].map(([m, label]) =>
    el("button", { type: "button", role: "radio", "aria-checked": String(S.mode === m), class: S.mode === m ? "on" : "", onclick: () => { S.mode = m; fillModes(); } }, label)));
  fillModes();
  const go = btn(t("gui.launch"), { class: "primary", onclick: () => busy(go, async () => {
    try {
      await post("/api/configs/launch", { path: c.path, profile: c.profiles.length ? S.profile : "", mode: S.mode });
      toast(c.profiles.length ? t("gui.launched_profile", c.name, S.profile) : t("gui.launched", c.name));
    } catch (e) { failToast(e); }
  }) }, "play");
  return el("div", { class: "block" },
    el("div", { class: "eyebrow", style: "margin-bottom:8px" }, t("gui.launch_title")),
    el("div", { class: "btn-row" }, profileSeg(c, () => { if (S.sub !== "edit" && S.sub !== "shortcuts") rerender(); }), modes, el("span", { class: "spacer" }), go),
    el("div", { class: "hint", style: "margin-top:8px" }, t("gui.launch_hint")));
}

function overview(c) {
  const box = el("div", {}, el("span", { class: "spinner" }));
  get("/api/configs/check?" + q({ path: c.path, profile: c.profiles.length ? S.profile : "" }))
    .then((r) => box.replaceChildren(drawOverview(r.overview), el("details", { class: "raw" },
      el("summary", {}, t("gui.ov_text")), el("pre", { class: "plan" }, r.plan))))
    .catch((e) => box.replaceChildren(failBanner(e)));
  return el("div", {}, el("p", { class: "intro" }, c.profiles.length ? t("gui.overview_intro_profile", S.profile) : t("gui.overview_intro")), box);
}

// The folder a list of paths shares, to name a box once rather than
// repeat it on every line ("" when they share none).
function commonDir(paths) {
  if (!paths.length) return "";
  const parts = paths.map((p) => p.split("\\").slice(0, -1));
  let n = 0;
  while (parts.every((x) => n < x.length && x[n].toLowerCase() === parts[0][n].toLowerCase())) n++;
  return parts[0].slice(0, n).join("\\");
}
function rel(path, dir) { return dir && path.toLowerCase().startsWith(dir.toLowerCase() + "\\") ? path.slice(dir.length + 1) : path; }

// One pair of arrows: copied over before the session (amber, to the
// right), copied back after (grey, to the left); a missing source copies
// nothing, its arrow is dashed red.
function syncArrows(missing) {
  const s = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  s.setAttribute("viewBox", "0 0 100 18");
  s.setAttribute("preserveAspectRatio", "none");
  s.setAttribute("class", "arrows" + (missing ? " missing" : ""));
  s.setAttribute("aria-hidden", "true");
  s.innerHTML = '<path class="in" d="M2 4H96M91 0.5L97 4L91 7.5"/>' + (missing ? "" : '<path class="back" d="M98 14H4M9 10.5L3 14L9 17.5"/>');
  return s;
}

function drawOverview(o) {
  const out = el("div", { class: "overview" });
  if (o.sync.length) {
    const left = commonDir(o.sync.map((p) => p.source));
    const right = commonDir(o.sync.map((p) => p.target));
    const n = o.sync.length;
    const grid = el("div", { class: "sync", style: `grid-template-rows: auto repeat(${n}, auto)` });
    const side = (col, ic, title, dir) => [
      el("div", { class: "side-bg", style: `grid-column:${col};grid-row:1 / ${n + 2}` }),
      el("div", { class: "side-head", style: `grid-column:${col};grid-row:1` },
        el("div", { class: "eyebrow" }, icon(ic), title), dir ? el("div", { class: "mono dim", title: dir }, dir) : "")];
    add(grid, side(1, "user", o.profile ? t("gui.ov_profile", o.profile) : t("gui.ov_config"), left));
    add(grid, side(3, "play", t("gui.ov_program"), right));
    grid.append(el("div", { class: "legend", style: "grid-column:2;grid-row:1" },
      el("span", { class: "in" }, t("gui.ov_before")), el("span", { class: "back" }, t("gui.ov_after"))));
    o.sync.forEach((p, i) => {
      const row = i + 2;
      const cell = (col, path, dir, isDir, bad) => el("div", { class: "file" + (bad ? " bad" : ""), title: path, style: `grid-column:${col};grid-row:${row}` },
        icon(isDir ? "folder" : "file"), el("span", { class: "name" }, rel(path, dir) + (isDir ? "\\" : "")),
        bad ? el("span", { class: "pill bad" }, t("gui.ov_absent")) : "");
      grid.append(cell(1, p.source, left, p.sourceDir, p.missing),
        el("div", { class: "link", style: `grid-column:2;grid-row:${row}` }, syncArrows(p.missing)),
        cell(3, p.target, right, p.targetDir, false));
    });
    out.append(el("div", { class: "eyebrow", style: "margin-bottom:8px" }, t("gui.ov_sync")), grid);
  }
  if (o.missing) out.append(failBanner(new Error(t("gui.ov_missing", o.missing))));

  const actions = (list) => list.map((a) => el("li", { class: a.missing ? "bad" : "", title: a.kind === "exec" ? a.args.join(" ") : a.kind === "delete" ? a.path : a.from + " → " + a.to },
    icon({ copy: "copy", delete: "trash", exec: "terminal" }[a.kind]),
    el("span", {}, a.kind === "exec" ? a.args.join(" ") : a.kind === "delete" ? base(a.path) : base(a.from) + " → " + base(a.to))));
  const step = (n, title, items, cls) => el("div", { class: "step" + (cls ? " " + cls : "") },
    el("div", { class: "eyebrow" }, n + " · " + title), items.length ? el("ul", {}, items) : el("p", { class: "hint" }, t("gui.ov_nothing")));
  const syncLine = (key) => o.sync.length ? [el("li", {}, icon("copy"), el("span", {}, t(key, o.sync.length)))] : [];
  const session = [
    el("li", { title: [o.command, ...(o.args || [])].join(" ") }, icon("play"), el("span", {}, [base(o.command), ...(o.args || [])].join(" "))),
    o.workdir && el("li", { title: o.workdir }, icon("folder"), el("span", {}, t("gui.ov_workdir", o.workdir))),
    o.follows !== base(o.command) && el("li", {}, icon("eye"), el("span", {}, t("gui.ov_follows", o.follows))),
    el("li", {}, icon("alert"), el("span", {}, o.ifRunning === "kill" ? t("gui.ov_if_kill") : t("gui.ov_if_abort"))),
    o.ready && el("li", { title: o.ready.file }, icon("check"), el("span", {}, t("gui.ov_ready", base(o.ready.file), o.ready.timeout))),
    o.shared && el("li", {}, icon("info"), el("span", {}, t("gui.ov_shared"))),
    el("li", { title: o.logDir }, icon("log"), el("span", {}, t("gui.ov_logs", o.logKeep))),
  ].filter(Boolean);
  out.append(el("div", { class: "steps" },
    step(1, t("gui.ov_step_before"), [...syncLine("gui.ov_copy_in"), ...actions(o.before)]),
    step(2, t("gui.ov_step_session"), session, "now"),
    step(3, t("gui.ov_step_after"), [...syncLine("gui.ov_copy_back"), ...actions(o.after)])));
  return out;
}

function logsTab(c) {
  const out = el("div", {}, el("span", { class: "spinner" }));
  const viewer = el("div");
  const profile = c.profiles.length ? S.profile : "";
  const openFile = async (f, item) => {
    for (const n of out.querySelectorAll(".list-item")) n.classList.toggle("on", n === item);
    try {
      const r = await get("/api/configs/logfile?" + q({ path: c.path, profile, file: f.name }));
      const box = el("div", { class: "log", style: "max-height:520px" });
      for (const line of r.text.split(/\r?\n/)) {
        if (!line) continue;
        const m = line.match(/^\S+ (\S+) (INFO|WARN|ERR |OUT )\s?(.*)$/);
        box.append(m ? el("div", { class: m[2].trim() }, el("span", { class: "t" }, m[1].slice(0, 8) + " "), m[3]) : el("div", {}, line));
      }
      viewer.replaceChildren(box);
    } catch (e) { failToast(e); }
  };
  get("/api/configs/logs?" + q({ path: c.path, profile })).then((files) => {
    if (!files.length) { out.replaceChildren(el("p", { class: "hint" }, t("gui.no_log"))); return; }
    const items = files.map((f) => el("div", { class: "list-item", tabindex: "0", onclick: (e) => openFile(f, e.currentTarget) },
      icon("file"), el("div", { class: "grow" }, el("div", { class: "title" }, fmtDate(f.modified)), el("div", { class: "sub mono" }, f.name)),
      el("span", { class: "hint" }, fmtSize(f.size))));
    out.replaceChildren(el("div", { class: "block", style: "padding:6px" }, items));
    items[0].click();
  }).catch((e) => out.replaceChildren(failBanner(e)));
  return el("div", {}, el("p", { class: "intro" }, profile ? t("gui.logs_intro_profile", profile) : t("gui.logs_intro")), out, viewer);
}

// shortcutsTab makes the shortcuts, asking first what the config lacks.
function shortcutsTab(c) {
  const box = el("div", {}, el("span", { class: "spinner" }));
  get("/api/configs/shortcuts?" + q({ path: c.path })).then((plan) => {
    const parts = [];
    let mode = c.mode || "run", ui = "tray";
    const folders = new Set(["${config_dir}"]);
    let other = null;
    const seg = (options, current, set) => {
      const s = el("div", { class: "seg" });
      const fill = () => s.replaceChildren(...options.map(([v, l]) => el("button", { type: "button", class: current() === v ? "on" : "", onclick: () => { set(v); fill(); } }, l)));
      fill();
      return s;
    };
    if (!plan.hasSection) {
      other = el("input", { type: "text", class: "mono", placeholder: t("gui.folder_path") });
      const otherRow = withBrowse(other, "folder");
      otherRow.hidden = true;
      const checks = plan.folders.Options.map((o) => {
        const cb = el("input", { type: "checkbox", checked: folders.has(o.Value) });
        const row = el("label", { class: "check" + (cb.checked ? " on" : "") }, cb, el("span", {}, o.Label));
        cb.addEventListener("change", () => {
          cb.checked ? folders.add(o.Value) : folders.delete(o.Value);
          row.classList.toggle("on", cb.checked);
          if (o.Value === "other") otherRow.hidden = !cb.checked;
        });
        return row;
      });
      parts.push(el("div", { class: "banner" }, icon("info"), el("span", { class: "grow" }, t("gui.no_section"))),
        field(t("gui.shortcut_mode"), "shortcuts.mode", t("gui.shortcut_mode_hint"), seg([["run", t("gui.mode_run")], ["watch", t("gui.mode_watch")]], () => mode, (v) => { mode = v; })),
        field(plan.folders.Prompt, "shortcuts.folder", t("gui.shortcut_folders_hint"), el("div", { class: "rows" }, checks, otherRow)));
    }
    if (!plan.hasUI) {
      parts.push(field(t("gui.ui"), "ui", t("gui.ui_hint"), seg([["tray", t("gui.ui_tray")], ["console", t("gui.ui_console")]], () => ui, (v) => { ui = v; })));
    }
    const icons = {};
    for (const p of plan.profiles) {
      const label = t("gui.icon_of", p.name || c.name);
      if (p.current) {
        parts.push(field(label, "icon", t("gui.icon_set"),
          el("div", { class: "row-line" }, el("img", { class: "thumb", src: "/api/icon?" + q({ path: p.current }), alt: "" }), el("span", { class: "mono" }, p.current))));
        continue;
      }
      const preview = el("img", { class: "thumb", alt: "", hidden: !p.suggest, src: p.suggest ? "/api/icon?" + q({ path: p.suggest }) : "" });
      const sel = el("select", {}, el("option", { value: "" }, t("gui.no_icon")),
        plan.icons.map((o) => el("option", { value: o.Value, selected: o.Value === p.suggest }, o.Label)));
      sel.addEventListener("change", () => { preview.hidden = !sel.value; if (sel.value) preview.src = "/api/icon?" + q({ path: sel.value }); });
      icons[p.name] = sel;
      parts.push(field(label, "icon", plan.icons.length ? t("gui.icon_hint") : t("gui.icon_none_found"), el("div", { class: "row-line" }, preview, sel)));
    }
    const result = el("div");
    const make = btn(t("gui.create_shortcuts"), { class: "primary", onclick: () => busy(make, async () => {
      const body = { path: c.path, icons: {} };
      if (!plan.hasSection) {
        body.mode = mode;
        body.folders = [...folders].filter((f) => f !== "other");
        if (folders.has("other") && other.value.trim()) body.folders.push(other.value.trim());
      }
      if (!plan.hasUI) body.ui = ui;
      for (const [p, sel] of Object.entries(icons)) body.icons[p] = sel.value;
      try {
        const r = await post("/api/configs/shortcuts", body);
        const bad = r.written.filter((w) => w.error);
        result.replaceChildren(
          el("div", { class: "banner " + (bad.length ? "bad" : "ok") }, icon(bad.length ? "alert" : "check"),
            el("span", { class: "grow" }, bad.length ? t("gui.shortcuts_some_failed", bad.length) : t("gui.shortcuts_done", r.written.length))),
          el("div", { class: "rows" }, r.written.map((w) => el("div", { class: "row-line" }, icon(w.error ? "alert" : "link"),
            el("span", { class: "grow mono" }, w.path), w.error ? el("span", { class: "hint" }, w.error) : w.kept ? el("span", { class: "hint" }, t("gui.kept_old")) : ""))),
          r.startMenu ? el("p", { class: "hint" }, t("gui.taskbar_hint")) : "");
        refreshConfigsLater();
      } catch (e) { failToast(e); }
    }) }, "link");
    box.replaceChildren(el("p", { class: "intro" }, t("gui.shortcuts_intro")), ...parts, el("div", { class: "btn-row" }, make), result);
  }).catch((e) => box.replaceChildren(failBanner(e)));
  return box;
}

// ---- editing a config, by forms ---------------------------------------------------

// Every field: key, kind, label, help, options (for a choice: [value, label]).
const FIELDS = () => [
  [t("gui.sec_program"), [
    ["command", "path", t("gui.f_command"), t("gui.h_command"), "program"],
    ["args", "list", t("gui.f_args"), t("gui.h_args")],
    ["workdir", "path", t("gui.f_workdir"), t("gui.h_workdir"), "folder"],
    ["track", "text", t("gui.f_track"), t("gui.h_track")],
    ["kill_on_exit", "list", t("gui.f_kill_on_exit"), t("gui.h_kill_on_exit")],
    ["if_running", "select", t("gui.f_if_running"), t("gui.h_if_running"), [["", t("gui.o_abort")], ["kill", t("gui.o_kill")]]],
    ["env", "map", t("gui.f_env"), t("gui.h_env")],
  ]],
  [t("gui.sec_display"), [
    ["name", "text", t("gui.f_name"), t("gui.h_name")],
    ["ui", "select", t("gui.f_ui"), t("gui.h_ui"), [["", t("gui.ui_console")], ["tray", t("gui.ui_tray")]]],
    ["icon", "icon", t("gui.f_icon"), t("gui.h_icon")],
    ["capture", "bool", t("gui.f_capture"), t("gui.h_capture")],
    ["lang", "select", t("gui.f_lang"), t("gui.h_lang"), [["", t("gui.o_lang_default")], ["fr", "Français"], ["en", "English"]]],
  ]],
  [t("gui.sec_vars"), [["vars", "map", t("gui.f_vars"), t("gui.h_vars")]]],
  [t("gui.sec_files"), [
    ["hooks.sync", "list", t("gui.f_sync"), t("gui.h_sync")],
    ["hooks.before", "actions", t("gui.f_before"), t("gui.h_before")],
    ["hooks.after", "actions", t("gui.f_after"), t("gui.h_after")],
    ["hooks.timeout", "text", t("gui.f_timeout"), t("gui.h_timeout")],
    ["backup", "bool", t("gui.f_backup"), t("gui.h_backup")],
  ]],
  [t("gui.sec_shortcuts"), [
    ["shortcuts.mode", "select", t("gui.f_sc_mode"), t("gui.h_sc_mode"), [["", t("gui.o_none")], ["run", t("gui.mode_run")], ["watch", t("gui.mode_watch")]]],
    ["shortcuts.name", "text", t("gui.f_sc_name"), t("gui.h_sc_name")],
    ["shortcuts.folder", "paths", t("gui.f_sc_folder"), t("gui.h_sc_folder"), "folder"],
  ]],
  [t("gui.sec_watch"), [
    ["watch.restart", "select", t("gui.f_restart"), t("gui.h_restart"), [["", t("gui.o_always")], ["on-failure", t("gui.o_on_failure")], ["never", t("gui.o_never")]]],
    ["watch.delay", "text", t("gui.f_delay"), t("gui.h_delay")],
    ["watch.max_delay", "text", t("gui.f_max_delay"), t("gui.h_max_delay")],
    ["watch.max_restarts", "number", t("gui.f_max_restarts"), t("gui.h_max_restarts")],
  ]],
  [t("gui.sec_ready"), [
    ["ready.file", "path", t("gui.f_ready_file"), t("gui.h_ready_file"), "save"],
    ["ready.timeout", "text", t("gui.f_ready_timeout"), t("gui.h_ready_timeout")],
  ]],
  [t("gui.sec_log"), [
    ["log.dir", "path", t("gui.f_log_dir"), t("gui.h_log_dir"), "folder"],
    ["log.keep", "number", t("gui.f_log_keep"), t("gui.h_log_keep")],
    ["exit.delay", "text", t("gui.f_exit_delay"), t("gui.h_exit_delay")],
    ["exit.wait_key", "select", t("gui.f_wait_key"), t("gui.h_wait_key"), [["", t("gui.o_on_error")], ["never", t("gui.o_never")], ["always", t("gui.o_always")]]],
  ]],
];

function getPath(doc, path) { return path.split(".").reduce((o, k) => (o && typeof o === "object" ? o[k] : undefined), doc); }
function setPath(doc, path, value) {
  const keys = path.split(".");
  let o = doc;
  for (const k of keys.slice(0, -1)) { if (!o[k] || typeof o[k] !== "object") o[k] = {}; o = o[k]; }
  o[keys[keys.length - 1]] = value;
}
function prune(v) {
  if (Array.isArray(v)) { const a = v.map(prune).filter((x) => x !== undefined); return a.length ? a : undefined; }
  if (v && typeof v === "object") {
    const o = {};
    for (const [k, x] of Object.entries(v)) { const p = prune(x); if (p !== undefined) o[k] = p; }
    return Object.keys(o).length ? o : undefined;
  }
  return v === "" || v === null ? undefined : v;
}
// splitArgs cuts a command line's arguments, keeping "quoted parts" whole;
// inside quotes \" is a quote and \\ a backslash, elsewhere a backslash is
// itself (C:\dir). joinArgs writes them back so that splitArgs gives the
// same arguments, quotes and empty ones included.
function splitArgs(s) {
  const out = [];
  let cur = null, quoted = false;
  for (let i = 0; i < s.length; i++) {
    const c = s[i];
    if (quoted && c === "\\" && (s[i + 1] === '"' || s[i + 1] === "\\")) { cur += s[++i]; continue; }
    if (c === '"') { quoted = !quoted; cur = cur ?? ""; continue; }
    if (!quoted && /\s/.test(c)) { if (cur !== null) out.push(cur); cur = null; continue; }
    cur = (cur ?? "") + c;
  }
  if (cur !== null) out.push(cur);
  return out;
}
function joinArgs(a) {
  return a.map((x) => (x === "" || /[\s"]/.test(x) ? '"' + x.replace(/\\/g, "\\\\").replace(/"/g, '\\"') + '"' : x)).join(" ");
}

// widget returns [element, read()]; every change calls dirty().
function widget(kind, value, options, dirty) {
  const watch = (e) => { e.addEventListener("input", dirty); e.addEventListener("change", dirty); return e; };
  const listOf = (items, make, addLabel) => {
    const box = el("div", { class: "rows" });
    const rows = [];
    const addB = btn(addLabel || t("gui.add"), { class: "text small", onclick: () => { addRow(); dirty(); } }, "plus");
    const addRow = (v) => {
      const [node, read] = make(v);
      const r = { read };
      const line = el("div", { class: "row-line" }, el("div", { class: "grow" }, node),
        iconBtn("trash", t("gui.remove"), () => { line.remove(); rows.splice(rows.indexOf(r), 1); dirty(); }));
      rows.push(r);
      box.insertBefore(line, addB);
    };
    box.append(addB);
    items.forEach(addRow);
    return [box, () => rows.map((r) => r.read()).filter((x) => x !== undefined && x !== "")];
  };
  switch (kind) {
    case "text": {
      const i = watch(el("input", { type: "text", class: "mono", value: value ?? "" }));
      return [i, () => i.value.trim()];
    }
    case "path": {
      const i = watch(el("input", { type: "text", class: "mono", value: value ?? "" }));
      return [withBrowse(i, options), () => i.value.trim()];
    }
    case "paths": {
      const items = Array.isArray(value) ? value : value ? [value] : [];
      return listOf(items.map(String), (v) => { const i = watch(el("input", { type: "text", class: "mono", value: v ?? "" })); return [withBrowse(i, options), () => i.value.trim()]; });
    }
    case "icon": {
      const i = watch(el("input", { type: "text", class: "mono", value: value ?? "" }));
      const img = el("img", { class: "thumb", alt: "", hidden: !value, src: value ? "/api/icon?" + q({ path: value }) : "" });
      i.addEventListener("change", () => { img.hidden = !i.value; if (i.value) img.src = "/api/icon?" + q({ path: i.value }); });
      return [el("div", { class: "row-line" }, img, el("div", { class: "grow" }, i), browse(i, "icon")), () => i.value.trim()];
    }
    case "number": {
      const i = watch(el("input", { type: "text", value: value ?? "", inputmode: "numeric", style: "max-width:140px" }));
      return [i, () => (i.value.trim() === "" ? "" : Number(i.value))];
    }
    case "bool": {
      const s = watch(el("select", {}, el("option", { value: "" }, t("gui.o_default")),
        el("option", { value: "true", selected: value === true }, t("gui.yes")), el("option", { value: "false", selected: value === false }, t("gui.no"))));
      return [s, () => (s.value === "" ? "" : s.value === "true")];
    }
    case "select": {
      const s = watch(el("select", {}, options.map(([v, label]) => el("option", { value: v, selected: String(value ?? "") === v }, label))));
      return [s, () => s.value];
    }
    case "list": {
      const items = Array.isArray(value) ? value : value ? [value] : [];
      return listOf(items.map(String), (v) => { const i = watch(el("input", { type: "text", class: "mono", value: v ?? "" })); return [i, () => i.value.trim()]; });
    }
    case "map": {
      const [box, read] = listOf(Object.entries(value || {}), (kv) => {
        const k = watch(el("input", { type: "text", class: "mono", value: kv?.[0] ?? "", placeholder: t("gui.name") }));
        const v = watch(el("input", { type: "text", class: "mono", value: kv ? String(kv[1]) : "", placeholder: t("gui.value") }));
        return [el("div", { class: "row-line" }, el("div", { style: "width:34%" }, k), el("div", { class: "grow" }, v)), () => (k.value.trim() ? [k.value.trim(), v.value] : undefined)];
      });
      return [box, () => Object.fromEntries(read())];
    }
    case "actions": {
      return listOf(value || [], (a) => {
        a = a || {};
        let type = a.copy ? "copy" : a.exec ? "exec" : "delete";
        const sel = el("select", { style: "min-width:130px" }, [["copy", t("gui.a_copy")], ["delete", t("gui.a_delete")], ["exec", t("gui.a_exec")]]
          .map(([k, l]) => el("option", { value: k, selected: k === type }, l)));
        const from = watch(el("input", { type: "text", class: "mono", value: a.copy?.from ?? a.delete ?? (a.exec ? a.exec[0] : "") }));
        const to = watch(el("input", { type: "text", class: "mono", value: a.copy?.to ?? "", placeholder: t("gui.a_to") }));
        const shown = a.exec ? joinArgs(a.exec.slice(1)) : "";
        const args = watch(el("input", { type: "text", class: "mono", value: shown, placeholder: t("gui.a_args") }));
        const place = () => {
          from.placeholder = { copy: t("gui.a_from"), delete: t("gui.a_path"), exec: t("gui.a_command") }[type];
          to.hidden = type !== "copy"; args.hidden = type !== "exec";
        };
        sel.addEventListener("change", () => { type = sel.value; place(); dirty(); });
        place();
        const fromB = browse(from, () => (type === "exec" ? "program" : "file"));
        const toB = browse(to, "save");
        const showTo = () => { toB.hidden = type !== "copy"; };
        sel.addEventListener("change", showTo);
        showTo();
        return [el("div", { class: "row-line" }, sel, el("div", { class: "grow" }, from), fromB, el("div", { class: "grow" }, to, args), toB), () => {
          const v = from.value.trim();
          if (!v) return undefined;
          if (type === "copy") return { copy: { from: v, to: to.value.trim() } };
          // Arguments left as they were are kept as they were, to the letter.
          if (type === "exec") return { exec: [v, ...(a.exec && args.value === shown ? a.exec.slice(1) : splitArgs(args.value))] };
          return { delete: v };
        }];
      }, t("gui.add_action"));
    }
  }
}

function editor(c) {
  const box = el("div", {}, el("span", { class: "spinner" }));
  get("/api/configs/edit?" + q({ path: c.path })).then((r) => {
    const doc = r.doc || {};
    const dirtyPill = el("span", { class: "pill warn", hidden: true }, t("gui.unsaved"));
    const dirty = () => { S.dirty = true; dirtyPill.hidden = false; };
    const readers = [];
    const sections = FIELDS().map(([title, fields]) => el("fieldset", { class: "section" }, el("legend", {}, title),
      fields.map(([key, kind, label, hint, options]) => {
        const [w, read] = widget(kind, getPath(doc, key), options, dirty);
        readers.push([key, read]);
        return field(label, key, hint, w);
      })));
    // Profiles: from what each had, so the keys the form does not show stay.
    const profilesBox = el("div", { class: "rows" });
    const profiles = [];
    const addProfile = (name, p) => {
      p = p || {};
      const nameI = el("input", { type: "text", value: name ?? "", style: "max-width:240px", placeholder: t("gui.profile_name") });
      nameI.addEventListener("input", dirty);
      const parts = [["icon", "icon", t("gui.f_icon")], ["vars", "map", t("gui.f_vars")], ["hooks.sync", "list", t("gui.f_sync")],
        ["hooks.before", "actions", t("gui.f_before")], ["hooks.after", "actions", t("gui.f_after")]].map(([k, kind, label]) => {
        const [w, read] = widget(kind, getPath(p, k), null, dirty);
        return [k, field(label, k, null, w), read];
      });
      const entry = { nameI, parts, orig: p };
      const block = el("div", { class: "block" },
        el("div", { class: "row-line", style: "margin-bottom:10px" }, el("span", { class: "eyebrow" }, t("gui.profile")), nameI, el("span", { class: "spacer" }),
          btn(t("gui.remove_profile"), { class: "text small danger", onclick: () => { block.remove(); profiles.splice(profiles.indexOf(entry), 1); dirty(); } }, "trash")),
        parts.map(([, node]) => node));
      profiles.push(entry);
      profilesBox.append(block);
    };
    for (const [n, p] of Object.entries(doc.profiles || {})) addProfile(n, p);
    const error = el("div");
    const save = btn(t("gui.save"), { class: "primary", onclick: () => busy(save, async () => {
      const out = {};
      const shown = new Set(FIELDS().flatMap(([, f]) => f.map(([k]) => k.split(".")[0])).concat(["profiles"]));
      for (const [k, v] of Object.entries(doc)) if (!shown.has(k)) out[k] = v;
      for (const [key, read] of readers) setPath(out, key, read());
      const ps = {};
      for (const { nameI, parts, orig } of profiles) {
        const n = nameI.value.trim();
        if (!n) continue;
        const p = JSON.parse(JSON.stringify(orig));
        for (const [k, , read] of parts) setPath(p, k, read());
        ps[n] = prune(p) || {};
      }
      const cleaned = prune(out) || {};
      if (Object.keys(ps).length) cleaned.profiles = ps;
      try {
        await post("/api/configs/edit", { path: c.path, doc: cleaned });
        S.dirty = false; dirtyPill.hidden = true; error.replaceChildren();
        toast(t("gui.saved"));
        refreshConfigsLater();
      } catch (e) {
        error.replaceChildren(failBanner(new Error(t("gui.save_refused", e.message))));
        error.scrollIntoView({ behavior: "smooth", block: "center" });
      }
    }) }, "check");
    const cancel = btn(t("gui.cancel"), { class: "text", onclick: () => { S.dirty = false; rerender(); } });
    box.replaceChildren(
      el("p", { class: "intro" }, t("gui.edit_hint")), error,
      ...sections,
      el("fieldset", { class: "section" }, el("legend", {}, t("gui.sec_profiles")), el("p", { class: "hint" }, t("gui.profiles_hint")),
        profilesBox, btn(t("gui.add_profile"), { class: "text small", onclick: () => { addProfile("", {}); dirty(); } }, "plus")),
      el("div", { class: "savebar" }, save, cancel, dirtyPill));
  }).catch((e) => box.replaceChildren(failBanner(e)));
  return box;
}

// ---- wizard -------------------------------------------------------------------------

const KIND = { text: 0, path: 1, yesno: 2, choice: 3, list: 4, multi: 5 };
const ACT = { recover: 0, save: 1, learn: 2, restore: 3, apply: 4, discard: 5 };
const CASE_ICONS = { dcs: "gamepad", game: "gamepad", watch: "pulse", backup: "archive" };
let wizardLog = null; // the log of the step running, kept across polls

async function viewWizard() {
  stopLive();
  let v;
  try { v = await get("/api/wizard"); } catch (e) { main(el("div", { class: "page" }, failBanner(e))); return; }
  const page = el("div", { class: "page" }, el("div", { class: "page-head" }, el("h2", {}, t("gui.wizard"))));
  main(page);
  if (!v.active) {
    const path = el("input", { type: "text", class: "mono", placeholder: t("gui.wizard_path") });
    const start = btn(t("gui.wizard_start"), { class: "primary", onclick: () => busy(start, async () => {
      try { await post("/api/wizard/start", { path: path.value }); viewWizard(); } catch (e) { failToast(e); }
    }) }, "wand");
    page.append(el("div", { class: "card" },
      el("p", { class: "intro" }, t("gui.wizard_intro")),
      el("ul", { class: "intro" }, el("li", {}, t("gui.wizard_point_a")), el("li", {}, t("gui.wizard_point_b")), el("li", {}, t("gui.wizard_point_c"))),
      el("details", {}, el("summary", { class: "hint", style: "cursor:pointer;margin-bottom:8px" }, t("gui.wizard_path_title")),
        field(t("gui.config_file"), null, t("gui.wizard_path_hint"), withBrowse(path, "newyaml"))),
      el("div", { class: "btn-row", style: "margin-top:12px" }, start)));
    return;
  }
  const answers = el("aside", { class: "card answers" }, el("div", { class: "eyebrow" }, t("gui.your_answers")),
    v.answers.length ? el("ol", {}, v.answers.map((a) => el("li", {}, el("div", { class: "q" }, a.prompt), el("div", { class: "a" }, a.answer || "—"))))
      : el("p", { class: "hint" }, t("gui.no_answer_yet")));
  const abandon = !v.done && !v.busy && btn(t("gui.wizard_cancel"), { class: "text small danger", onclick: async () => {
    if (!(await confirmBox(t("gui.wizard_cancel"), t("gui.ask_close_wizard"), t("gui.wizard_cancel"), true))) return;
    try { await post("/api/wizard/close"); toast(t("gui.wizard_abandoned")); viewWizard(); } catch (e) { failToast(e); }
  } }, "x");
  const card = el("div", { class: "card question" });
  page.append(el("div", { class: "wizard" }, el("div", {}, card, abandon), answers));
  wizardStep(card, v);
}

function wizardStep(card, v) {
  if (v.done) {
    wizardLog = null;
    card.append(
      v.applied ? el("div", { class: "banner ok" }, icon("check"), el("span", { class: "grow" }, t("gui.wizard_created", v.config)))
        : el("div", { class: "banner" }, icon("info"), el("span", { class: "grow" }, t("gui.wizard_nothing"))),
      v.written?.length ? el("div", { class: "rows" }, el("div", { class: "eyebrow" }, t("gui.shortcuts")),
        v.written.map((w) => el("div", { class: "row-line" }, icon(w.error ? "alert" : "link"), el("span", { class: "grow mono" }, w.path), w.error && el("span", { class: "hint" }, w.error)))) : "",
      v.startMenu ? el("p", { class: "hint" }, t("gui.taskbar_hint")) : "",
      el("div", { class: "btn-row", style: "margin-top:14px" },
        v.applied && btn(t("gui.see_config"), { class: "primary", onclick: async () => { await post("/api/wizard/close"); S.config = v.config; show("configs"); } }, "folder"),
        btn(t("gui.wizard_again"), { onclick: async () => { await post("/api/wizard/close"); viewWizard(); } }, "wand")));
    return;
  }
  const step = v.step;
  if (!v.busy && !step.Action) { wizardLog = null; questionCard(card, step, v); return; }
  const learn = step.Action && step.Action.Kind === ACT.learn;
  const logBox = wizardLog || (wizardLog = el("div", { class: "log" }));
  if (v.error) {
    card.append(failBanner(new Error(v.error)),
      el("div", { class: "btn-row" },
        v.running && btn(t("gui.stop_game", v.running.Name), { class: "primary", onclick: (e) => busy(e.currentTarget, () => post("/api/wizard/stopgame").then(viewWizard).catch(failToast)) }, "stop"),
        btn(t("gui.retry"), { onclick: () => post("/api/wizard/run").then(viewWizard).catch(failToast) }, "restart")),
      logBox.childNodes.length ? logBox : "");
    return;
  }
  if (learn && !v.busy) {
    card.append(el("div", { class: "status-panel" }, el("div", { style: "color:var(--amber)" }, icon("gamepad")),
      el("div", {}, el("h3", {}, t("gui.learn_title", step.Action.Profile)), el("p", { class: "help" }, step.Action.Message),
        btn(t("gui.start_game"), { class: "primary", onclick: (e) => busy(e.currentTarget, () => post("/api/wizard/run").then(viewWizard).catch(failToast)) }, "play"))));
    return;
  }
  if (!v.busy) { post("/api/wizard/run").then(viewWizard).catch(failToast); return; }
  const titles = { [ACT.recover]: t("gui.act_recover"), [ACT.save]: t("gui.act_save"), [ACT.restore]: t("gui.act_restore"), [ACT.apply]: t("gui.act_apply"), [ACT.discard]: t("gui.act_discard") };
  card.append(el("div", { class: "status-panel" }, el("span", { class: "spinner big" }),
    el("div", { class: "grow" }, el("h3", {}, learn ? t("gui.playing", step.Action.Profile) : titles[step.Action?.Kind] || t("gui.wizard_busy")),
      learn ? el("p", { class: "help" }, t("gui.playing_hint")) : "")), logBox);
  const es = new EventSource("/api/wizard/events");
  es.onmessage = (ev) => logLine(logBox, JSON.parse(ev.data));
  live.push(es);
  live.push(setInterval(async () => {
    try { const nv = await get("/api/wizard"); if (!nv.busy) viewWizard(); } catch {}
  }, 800));
}

function questionCard(card, step, v) {
  const qn = step.Question;
  const def = qn.Default || [];
  let read;
  const error = el("div");
  const next = btn(t("gui.next"), { class: "primary", onclick: () => submit() });
  next.append(icon("right"));
  const submit = () => busy(next, async () => {
    try { await post("/api/wizard/answer", { values: read() }); viewWizard(); }
    catch (e) { error.replaceChildren(failBanner(e)); }
  });
  const content = [];
  if (step.ID === "case") {
    let chosen = def[0] || qn.Options[0].Value;
    // Marked in place rather than redrawn, so the focus stays and Enter goes on.
    const mark = () => [...tiles.children].forEach((b) => { const on = b.dataset.value === chosen; b.classList.toggle("on", on); b.setAttribute("aria-checked", String(on)); });
    const tiles = el("div", { class: "tiles", role: "radiogroup" }, qn.Options.map((o) => el("button", { type: "button", role: "radio", class: "tile", "data-value": o.Value,
      onclick: () => { chosen = o.Value; mark(); }, ondblclick: () => { chosen = o.Value; submit(); } },
      icon(CASE_ICONS[o.Value] || "wand"), el("span", {}, o.Label))));
    mark();
    // The arrows move the choice, as in any group of radio buttons.
    tiles.addEventListener("keydown", (e) => {
      const step = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 }[e.key];
      if (!step) return;
      e.preventDefault();
      const all = [...tiles.children];
      const to = all[(all.indexOf(document.activeElement) + step + all.length) % all.length];
      to.focus();
      to.click();
    });
    setTimeout(() => tiles.querySelector(".on")?.focus());
    content.push(tiles);
    read = () => [chosen];
  } else if (qn.Kind === KIND.choice || qn.Kind === KIND.yesno) {
    let chosen = def.length ? def[0] : qn.Options[0].Value;
    const rows = el("div", { class: "rows", role: "radiogroup" }, qn.Options.map((o) => el("label", { class: "check" + (o.Value === chosen ? " on" : "") },
      el("input", { type: "radio", name: "choice", checked: o.Value === chosen, onchange: (e) => {
        chosen = o.Value;
        rows.querySelectorAll("label").forEach((l) => l.classList.toggle("on", l.contains(e.target)));
      } }), el("span", {}, o.Label))));
    setTimeout(() => rows.querySelector("input:checked")?.focus());
    content.push(rows);
    read = () => [chosen];
  } else if (qn.Kind === KIND.multi) {
    const review = step.ID.startsWith("review:");
    const picked = new Set(def);
    content.push(el("div", { class: "rows" }, qn.Options.map((o) => {
      const cb = el("input", { type: "checkbox", checked: picked.has(o.Value) });
      const diff = el("div");
      const label = el("label", { class: "check" + (picked.has(o.Value) ? " on" : "") }, cb, el("span", { class: "grow" }, o.Label));
      cb.addEventListener("change", () => { cb.checked ? picked.add(o.Value) : picked.delete(o.Value); label.classList.toggle("on", cb.checked); });
      return el("div", {}, el("div", { class: "row-line" }, el("div", { class: "grow" }, label),
        review && btn(t("gui.show_diff"), { class: "text small", onclick: () => toggleDiff(diff, o.Value) }, "eye")), diff);
    })));
    read = () => qn.Options.map((o) => o.Value).filter((x) => picked.has(x));
    setTimeout(() => card.querySelector("input")?.focus());
  } else if (qn.Kind === KIND.list) {
    const entries = [];
    const list = el("div", { class: "rows" });
    const input = el("input", { type: "text", placeholder: t("gui.list_placeholder") });
    const addOne = () => {
      const val = input.value.trim();
      if (!val) return false;
      entries.push(val);
      const line = el("div", { class: "row-line block", style: "padding:6px 10px" }, el("span", { class: "grow" }, val),
        iconBtn("x", t("gui.remove"), () => { entries.splice(entries.indexOf(val), 1); line.remove(); }));
      list.append(line);
      input.value = "";
      return true;
    };
    input.addEventListener("keydown", (e) => { if (e.key === "Enter") { e.preventDefault(); e.stopPropagation(); if (!addOne()) submit(); } });
    content.push(list, el("div", { class: "row-line" }, el("div", { class: "grow" }, input), qn.Pick && browse(input, qn.Pick, () => { addOne(); input.focus(); }), btn(t("gui.add"), { onclick: addOne }, "plus")),
      el("span", { class: "hint" }, t("gui.list_hint")));
    read = () => { addOne(); return entries; };
    setTimeout(() => input.focus());
  } else {
    const input = el("input", { type: "text", class: qn.Kind === KIND.path ? "mono" : "", placeholder: def[0] || "" });
    content.push(qn.Pick ? withBrowse(input, qn.Pick) : input);
    if (def[0]) content.push(el("span", { class: "hint" }, t("gui.empty_default", def[0])));
    read = () => [input.value];
    setTimeout(() => input.focus());
  }
  // Enter answers, from a field or from a choice; on any other button it presses that button.
  card.addEventListener("keydown", (e) => {
    if (e.key !== "Enter") return;
    const choice = e.target.getAttribute("role") === "radio";
    if (e.target.tagName === "BUTTON" && !choice) return;
    e.preventDefault();
    if (choice) e.target.click();
    submit();
  });
  const back = btn(t("gui.back"), { class: "text", disabled: !v.answers.length, onclick: () => post("/api/wizard/back").then(viewWizard).catch(failToast) }, "left");
  card.append(el("h3", {}, qn.Prompt), qn.Help ? el("p", { class: "help" }, qn.Help) : "", add(el("div", { class: "field" }), content), error,
    el("div", { class: "nav" }, back, next));
}

async function toggleDiff(box, file) {
  if (box.childNodes.length) { box.replaceChildren(); return; }
  let d;
  try { d = await get("/api/wizard/diff?" + q({ file })); } catch (e) { failToast(e); return; }
  if (d.binary) {
    box.append(el("p", { class: "hint" }, t("gui.diff_binary", d.before < 0 ? "—" : fmtSize(d.before), d.after < 0 ? "—" : fmtSize(d.after))));
    return;
  }
  const pre = el("div", { class: "log diff" });
  for (const op of d.ops || []) {
    const ch = String.fromCharCode(op.kind);
    if (ch === "~") { pre.append(el("div", { class: "skip" }, t("gui.diff_skipped", op.text))); continue; }
    pre.append(el("div", { class: { "+": "add", "-": "del" }[ch] || "" }, ({ "+": "+ ", "-": "− " }[ch] || "  ") + op.text));
  }
  box.append(pre);
}

// ---- start -----------------------------------------------------------------------

(async function start() {
  try { if (localStorage.getItem("pm-theme") === "light") document.documentElement.classList.add("light"); } catch {}
  // While this page is open, pm keeps the interface running.
  new EventSource("/api/alive");
  const i = await get("/api/i18n");
  M = i.messages;
  LANG = i.lang;
  document.documentElement.lang = i.lang;
  const tabs = [["sessions", t("gui.tab_sessions"), "pulse"], ["configs", t("gui.tab_configs"), "folder"], ["wizard", t("gui.tab_wizard"), "wand"]];
  document.getElementById("tabs").replaceChildren(...tabs.map(([k, label, ic]) =>
    el("button", { type: "button", "data-tab": k, onclick: () => show(k) }, icon(ic), label)));
  tools();
  window.addEventListener("beforeunload", (e) => { if (S.dirty) { e.preventDefault(); e.returnValue = ""; } });
  // An icon pm cannot read shows nothing rather than a broken image.
  document.addEventListener("error", (e) => { if (e.target.classList?.contains("thumb")) e.target.hidden = true; }, true);
  const h = location.hash.slice(1);
  show(["sessions", "configs", "wizard"].includes(h) ? h : "sessions");
})();
