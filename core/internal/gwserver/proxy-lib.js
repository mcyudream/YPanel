/*!
 * YPanel webgw proxy-lib.js —— 内网浏览器会话式反代的前端请求拦截库（自研，无依赖）。
 * 由网关注入到被代理的 text/html 文档 <head> 最前，先于页面任何脚本执行；
 * 把页面运行时发出的请求地址改写进会话前缀（/s/{sid}[/~h/<host>]），交由网关代理。
 *
 * 运行时配置由网关在库体之前渲染：
 *   window.__YP_WEBGW__ = { base: "/s/{sid}", sid: "...", target: "scheme://host:port" }
 * 刻意不拦截：history.pushState/replaceState（SPA 依赖 location.pathname 做路由，hook 反致错乱）、
 * window.location 赋值（语言级不可拦截）——这两类由网关根逃逸回捞兜底。
 */
(function () {
  'use strict';
  var CFG = window.__YP_WEBGW__;
  if (!CFG || !CFG.base || window.__YP_WEBGW_HOOKED__) return;
  window.__YP_WEBGW_HOOKED__ = true;

  // ---------- 本页前缀：第一跳页面 = CFG.base；跨 host 页面 = CFG.base + "/~h/<b64>" ----------
  var MY = { base: CFG.base, origin: CFG.target };

  function escRe(s) { return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); }
  (function () {
    var m = new RegExp('^' + escRe(CFG.base) + '(?:/~h/([^/]+))?').exec(location.pathname);
    if (!m) return;
    MY.base = m[0];
    if (m[1]) {
      try {
        MY.origin = atob(m[1].replace(/-/g, '+').replace(/_/g, '/'));
      } catch (e) { /* 解码失败按第一跳处理 */ }
    }
  })();

  // origin 归一：ws/wss 折算为 http/https，缺省端口补齐（与服务端 target 归一口径一致）
  function originOf(u) {
    var proto = u.protocol;
    if (proto === 'ws:') proto = 'http:';
    else if (proto === 'wss:') proto = 'https:';
    var def = proto === 'https:' ? '443' : '80';
    return proto + '//' + u.host + (u.port ? '' : ':' + def);
  }

  function b64url(s) {
    return btoa(s).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  }

  // ---------- URL 改写核心 ----------
  // 返回改写后的地址；无需改写时原样返回（引用相等即可判断）。
  function rewrite(raw) {
    if (raw == null) return raw;
    var s = String(raw);
    if (!s || s.charAt(0) === '#') return s; // 纯锚点/空串
    var u;
    try { u = new URL(s, location.href); } catch (e) { return raw; }
    var p = u.protocol;
    // 非 http(s)/ws(s) 一律放行：blob/data/about/javascript/mailto/tel 等
    if (p !== 'http:' && p !== 'https:' && p !== 'ws:' && p !== 'wss:') return raw;
    // 已在会话前缀内
    if (u.pathname === CFG.base || u.pathname.indexOf(CFG.base + '/') === 0) return raw;
    var rest = u.pathname + u.search + u.hash;
    var o = originOf(u);
    if (o === MY.origin) return MY.base + rest; // 本页所属目标：直接落本页前缀
    return CFG.base + '/~h/' + b64url(o) + rest; // 其它内网地址：跨 host 段经网关转发
  }

  // ---------- 网络原语 ----------
  var _fetch = window.fetch;
  if (_fetch) {
    window.fetch = function (input, init) {
      try {
        if (typeof Request !== 'undefined' && input instanceof Request) {
          var nu = rewrite(input.url);
          if (nu !== input.url) input = new Request(nu, input); // 失败即回退原请求
        } else {
          input = rewrite(input);
        }
      } catch (e) { /* 保持原样，交回捞兜底 */ }
      return _fetch.call(this, input, init);
    };
  }

  var _open = XMLHttpRequest.prototype.open;
  XMLHttpRequest.prototype.open = function (method, url) {
    var args = arguments;
    try {
      if (args.length > 1) args[1] = rewrite(url);
    } catch (e) { /* 原样 */ }
    return _open.apply(this, args);
  };

  function hookCtor(name, constants) {
    var Orig = window[name];
    if (!Orig) return;
    function Patched(url, a, b) {
      var u;
      try { u = rewrite(url); } catch (e) { u = url; }
      switch (arguments.length) {
        case 1: return new Orig(u);
        case 2: return new Orig(u, a);
        default: return new Orig(u, a, b);
      }
    }
    Patched.prototype = Orig.prototype;
    for (var k in constants) {
      try { Patched[k] = Orig[k]; } catch (e) { /* 忽略只读常量 */ }
    }
    window[name] = Patched;
  }
  hookCtor('WebSocket', { CONNECTING: 0, OPEN: 1, CLOSING: 2, CLOSED: 3 });
  hookCtor('EventSource', { CONNECTING: 0, OPEN: 1, CLOSED: 2 });
  hookCtor('Worker', null);

  if (navigator.sendBeacon) {
    var _beacon = navigator.sendBeacon.bind(navigator);
    navigator.sendBeacon = function (url, data) {
      try { url = rewrite(url); } catch (e) { /* 原样 */ }
      return _beacon(url, data);
    };
  }

  var _winOpen = window.open;
  window.open = function (url) {
    var u = url;
    try { u = rewrite(url); } catch (e) { /* 原样 */ }
    var args = [u];
    for (var i = 1; i < arguments.length; i++) args.push(arguments[i]);
    return _winOpen.apply(window, args);
  };

  // ---------- DOM 资源属性 ----------
  // IDL setter 钩子：覆盖 el.src = x 的动态资源插入（含 webpack/vite 动态 import 的 script 注入）。
  // 刻意不 hook HTMLAnchorElement.href：SPA 路由库读取链接做解析，改写会致错；点击导航由网关回捞。
  function hookProp(proto, prop) {
    var d = Object.getOwnPropertyDescriptor(proto, prop);
    if (!d || !d.set || !d.configurable) return;
    Object.defineProperty(proto, prop, {
      configurable: true,
      enumerable: d.enumerable,
      get: function () { return d.get.call(this); },
      set: function (v) {
        try { v = rewrite(v); } catch (e) { /* 原样 */ }
        return d.set.call(this, v);
      }
    });
  }
  [
    [window.HTMLScriptElement, 'src'],
    [window.HTMLImageElement, 'src'],
    [window.HTMLIFrameElement, 'src'],
    [window.HTMLSourceElement, 'src'],
    [window.HTMLTrackElement, 'src'],
    [window.HTMLEmbedElement, 'src'],
    [window.HTMLMediaElement, 'src'],
    [window.HTMLLinkElement, 'href'],
    [window.HTMLObjectElement, 'data'],
    [window.HTMLFormElement, 'action']
  ].forEach(function (it) { if (it[0]) hookProp(it[0].prototype, it[1]); });

  // setAttribute 钩子：按资源标签白名单（jQuery attr() 等路径）；SVG 元素 tagName 小写不命中，天然跳过。
  var RES_TAGS = { SCRIPT: 1, IMG: 1, IFRAME: 1, SOURCE: 1, TRACK: 1, EMBED: 1, LINK: 1, OBJECT: 1, FORM: 1, VIDEO: 1, AUDIO: 1 };
  var RES_ATTRS = { src: 1, href: 1, action: 1, data: 1, poster: 1 };
  var _setAttr = Element.prototype.setAttribute;
  Element.prototype.setAttribute = function (name, value) {
    try {
      if (typeof value === 'string' && RES_ATTRS[String(name).toLowerCase()] && RES_TAGS[this.tagName]) {
        value = rewrite(value);
      }
    } catch (e) { /* 原样 */ }
    return _setAttr.call(this, name, value);
  };

  // ---------- meta 形式 CSP 清理 ----------
  // 方案 A 只删响应头 CSP；<meta http-equiv> 下发的删不掉响应头，需在此同步移除，
  // 并观察至 DOMContentLoaded（CSP meta 正常只出现在静态 head 中，之后断开节省开销）。
  function killMetaCSP(node) {
    if (!node || node.nodeType !== 1) return;
    if (node.tagName === 'META' && /content-security-policy/i.test(node.getAttribute('http-equiv') || '')) {
      if (node.parentNode) node.parentNode.removeChild(node);
      return;
    }
    if (node.querySelectorAll) {
      var list = node.querySelectorAll('meta[http-equiv="Content-Security-Policy" i]');
      for (var i = 0; i < list.length; i++) {
        if (list[i].parentNode) list[i].parentNode.removeChild(list[i]);
      }
    }
  }
  killMetaCSP(document);
  if (window.MutationObserver) {
    var mo = new MutationObserver(function (muts) {
      for (var i = 0; i < muts.length; i++) {
        var added = muts[i].addedNodes;
        for (var j = 0; j < added.length; j++) killMetaCSP(added[j]);
      }
    });
    mo.observe(document.documentElement || document, { childList: true, subtree: true });
    document.addEventListener('DOMContentLoaded', function () { mo.disconnect(); }, { once: true });
  }
})();
