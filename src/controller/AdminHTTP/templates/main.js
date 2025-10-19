// ===== Global helpers (templates, utils, net, state) =====
function getTemplate(id) {
  var tpl = document.getElementById(id);
  return tpl && 'content' in tpl ? tpl : null;
}

function renderFromTemplate(id, fill) {
  var tpl = getTemplate(id);
  if (!tpl) return null;
  var node = document.importNode(tpl.content, true);
  if (typeof fill === 'function') fill(node);
  return node;
}

function qs(sel, root) { return (root || document).querySelector(sel); }
function qsa(sel, root) { return Array.prototype.slice.call((root || document).querySelectorAll(sel)); }
function fetchJson(url, options) {
  var opts = options || {};
  opts.headers = Object.assign({ 'Accept': 'application/json' }, opts.headers || {});
  return fetch(url, opts).then(function(resp){
    if (!resp.ok) throw new Error('http_' + resp.status);
    return resp.json().catch(function(){ return {}; });
  });
}

var api = {
  get: function(url, opts) { return fetchJson("{{APP_URL}}"+url, Object.assign({ method: 'GET' }, opts || {})); },
  post: function(url, body, opts) { return fetchJson("{{APP_URL}}"+url, Object.assign({ method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body || {}) }, opts || {})); },
  put: function(url, body, opts) { return fetchJson("{{APP_URL}}"+url, Object.assign({ method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body || {}) }, opts || {})); },
  del: function(url, opts) { return fetchJson("{{APP_URL}}"+url, Object.assign({ method: 'DELETE' }, opts || {})); }
};

var AppState = {
  get servicesCatalog() { return Array.isArray(window.__servicesCatalog) ? window.__servicesCatalog : []; },
  set servicesCatalog(list) { window.__servicesCatalog = Array.isArray(list) ? list : []; }
};

(function() {

  var overlay = document.getElementById('servicesOverlay');
  var toggleBtn = document.getElementById('toggleServicesBtn');
  var closeBtn = document.getElementById('closeServicesBtn');
  var featuresSection = document.querySelector('.features');
  var backdrop = document.getElementById('servicesBackdrop');

  if (!overlay || !toggleBtn) return;

  function isOpen() {
    return overlay.getAttribute('aria-hidden') === 'false';
  }

  function openOverlay() {
    overlay.setAttribute('aria-hidden', 'false');
    toggleBtn.setAttribute('aria-expanded', 'true');
  }

  function closeOverlay() {
    overlay.setAttribute('aria-hidden', 'true');
    toggleBtn.setAttribute('aria-expanded', 'false');
  }

  function toggleOverlay() {
    isOpen() ? closeOverlay() : openOverlay();
  }

  toggleBtn.addEventListener('click', toggleOverlay);
  if (closeBtn) closeBtn.addEventListener('click', closeOverlay);
  if (backdrop) backdrop.addEventListener('click', closeOverlay);

  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' && isOpen()) {
      closeOverlay();
    }
  });

  // Optional: Click outside to close when clicking on features backdrop area
  if (featuresSection) {
    featuresSection.addEventListener('click', function() {
      if (isOpen()) closeOverlay();
    });
  }
})();

// Services catalog loading + overlay rendering
(function() {
  var servicesListEl = document.getElementById('servicesList');

  function normalizeService(item) {
    var id = item && (item.id != null ? String(item.id) : '');
    var name = item && (item.name != null ? String(item.name) : id);
    var active = !!(item && item.active);
    return { id: id, name: name, active: active };
  }

  function renderServicesOverlay(list) {
    if (!servicesListEl) return;
    servicesListEl.innerHTML = '';
    (list || []).forEach(function(svc) {
      var frag = renderFromTemplate('serviceListItemTemplate', function(node){
        var root = node.firstElementChild || node;
        if (!root) return;
        root.setAttribute('data-service-id', svc.id || '');
        var nameEl = root.querySelector('.name');
        if (nameEl) nameEl.textContent = svc.name || svc.id || '';
        var badge = root.querySelector('.services-overlay__badge');
        if (badge) {
          badge.setAttribute('data-badge', svc.active ? 'active' : 'inactive');
          badge.textContent = svc.active ? 'Активен' : 'Неактивен';
        }
        var delBtn = root.querySelector('[data-action="delete"]');
        if (delBtn) delBtn.disabled = !!svc.active;
      });
      if (frag) servicesListEl.appendChild(frag);
    });
  }

  function populateServiceFilterDirect(list) {
    var selectEl = document.getElementById('serviceSelect');
    if (!selectEl) return;
    var prev = selectEl.value;
    selectEl.innerHTML = '';
    var allOpt = document.createElement('option');
    allOpt.value = '';
    allOpt.textContent = 'Все сервисы';
    selectEl.appendChild(allOpt);
    (list || []).forEach(function(svc) {
      var opt = document.createElement('option');
      opt.value = svc.id;
      opt.textContent = svc.name || svc.id;
      selectEl.appendChild(opt);
    });
    if (prev && (list || []).some(function(s){ return s.id === prev; })) {
      selectEl.value = prev;
    } else {
      selectEl.value = '';
    }
    try {
      var evt = document.createEvent('Event');
      evt.initEvent('change', true, true);
      selectEl.dispatchEvent(evt);
    } catch (e) {}
  }

  function afterLoadServices(list) {
    renderServicesOverlay(list);
    if (typeof window.__populateServiceOptions === 'function') {
      window.__populateServiceOptions();
    } else {
      populateServiceFilterDirect(list);
    }
  }

  function fetchServices() {
    try {
      api.get('/api/services')
        .then(function(arr){
          if (!Array.isArray(arr)) return;
          var norm = arr.map(normalizeService);
          window.__servicesCatalog = norm;
          afterLoadServices(norm);
        })
        .catch(function(){ /* ignore fetch errors for now */ });
    } catch (e) {
      // no-op
    }
  }

  if (!Array.isArray(window.__servicesCatalog)) {
    window.__servicesCatalog = [];
  }
  fetchServices();

  // Expose helper to refresh UI when catalog changes externally
  window.__refreshServicesUi = function() {
    try {
      var list = Array.isArray(window.__servicesCatalog) ? window.__servicesCatalog : [];
      renderServicesOverlay(list);
      if (typeof window.__populateServiceOptions === 'function') {
        window.__populateServiceOptions();
      } else {
        populateServiceFilterDirect(list);
      }
    } catch (e) {}
  };

  // Handle delete clicks with optimistic UI and server sync
  if (servicesListEl) {
    servicesListEl.addEventListener('click', function(e) {
      var btn = e.target && e.target.closest('button[data-action="delete"]');
      if (!btn || btn.disabled) return;
      var li = btn.closest('li');
      if (!li) return;
      var id = li.getAttribute('data-service-id') || '';
      if (!id) return;
      var nameNode = li.querySelector('.name');
      var name = nameNode ? String(nameNode.textContent || '').trim() : '';

      var ok = true;
      try { ok = window.confirm('Удалить сервис «' + (name || id) + '»?'); } catch (_) {}
      if (!ok) return;

      var originalHtml = btn.innerHTML;
      btn.innerHTML = 'Удаляем…';
      btn.disabled = true;
      btn.setAttribute('aria-busy', 'true');

      api.del('/api/services/' + encodeURIComponent(id))
        .then(function(){ return api.get('/api/services'); })
        .then(function(arr){
          var norm = Array.isArray(arr) ? arr.map(normalizeService) : [];
          window.__servicesCatalog = norm;
          if (typeof window.__refreshServicesUi === 'function') {
            window.__refreshServicesUi();
          }
        })
        .catch(function(){
          try { window.alert('Не удалось удалить сервис. Повторите попытку.'); } catch (_) {}
          // restore button state if still in DOM
          if (btn && btn.isConnected) {
            btn.innerHTML = originalHtml;
            btn.disabled = false;
            btn.removeAttribute('aria-busy');
          }
        });
    });
  }
})();

// Features list logic
(function() {
  var listEl = document.getElementById('featuresList');
  var featuresLoaderEl = document.getElementById('featuresLoader');
  var emptyEl = document.getElementById('featuresEmpty');
  var template = document.getElementById('featureItemTemplate');
  var addBtn = document.getElementById('openFeatureModalBtn');
  var featuresSection = document.querySelector('.features');

  // Controls
  var statusToggleEl = document.querySelector('.features__status-toggle');
  var viewToggleEl = document.querySelector('.features__view-toggle');
  var searchInputEl = document.getElementById('featureSearch');
  var serviceSelectEl = document.getElementById('serviceSelect');
  var paginationEl = document.getElementById('featuresPagination');
  var pageInfoEl = document.getElementById('featuresPageInfo');

  if (!listEl || !template) return;

  function nowIso() {
    return new Date().toISOString();
  }

  function formatDate(iso) {
    try {
      return new Date(iso).toLocaleString('ru-RU', {
        year: 'numeric', month: '2-digit', day: '2-digit',
        hour: '2-digit', minute: '2-digit'
      });
    } catch (e) {
      return iso;
    }
  }

  var features = [
  ];

  // UI State
  var currentStatus = 'all'; // all | attention
  var currentView = 'detailed'; // detailed | simple
  var currentQuery = '';
  var currentService = '';
  var currentPage = 1;
  var pageSize = 5;
  var serverTotalPages = 1; // reflects total pages reported by backend
  var featuresFetchController = null; // AbortController for in-flight features request
  var featuresFetchSeq = 0; // monotonically increasing request id

  // ===== Persistence (localStorage) =====
  var UI_STATE_STORAGE_KEY = 'fc_ui_state';

  function loadUiState() {
    try {
      var raw = window.localStorage && window.localStorage.getItem(UI_STATE_STORAGE_KEY);
      return raw ? JSON.parse(raw) : null;
    } catch (e) {
      return null;
    }
  }

  function saveUiState() {
    try {
      var payload = {
        currentStatus: currentStatus,
        currentView: currentView,
        currentQuery: currentQuery,
        currentService: currentService,
        currentPage: currentPage
      };
      if (window.localStorage) {
        window.localStorage.setItem(UI_STATE_STORAGE_KEY, JSON.stringify(payload));
      }
    } catch (e) {}
  }

  function applyInitialUiState() {
    var st = loadUiState();
    if (!st) return;

    if (st.currentStatus) currentStatus = st.currentStatus;
    if (st.currentView) currentView = st.currentView;
    if (typeof st.currentQuery === 'string') currentQuery = st.currentQuery;
    if (typeof st.currentService === 'string') currentService = st.currentService;
    if (typeof st.currentPage === 'number') currentPage = Math.max(1, st.currentPage | 0);

    // Apply pressed states and visual toggles
    if (statusToggleEl) {
      var sBtn = statusToggleEl.querySelector('button[data-status="' + currentStatus + '"]') || statusToggleEl.querySelector('button[data-status]');
      if (sBtn) setPressedInGroup(statusToggleEl, sBtn);
    }
    if (viewToggleEl) {
      var vBtn = viewToggleEl.querySelector('button[data-view="' + currentView + '"]') || viewToggleEl.querySelector('button[data-view]');
      if (vBtn) setPressedInGroup(viewToggleEl, vBtn);
    }
    if (featuresSection) {
      if (currentView === 'simple') {
        featuresSection.classList.add('features--simple');
      } else {
        featuresSection.classList.remove('features--simple');
      }
    }
    if (searchInputEl) {
      searchInputEl.value = currentQuery || '';
    }
  }

  function renderServices(ul, services) {
    ul.innerHTML = '';
    (services || []).forEach(function(svc) {
      var li = document.createElement('li');
      li.textContent = (svc && (svc.name || svc.id)) || String(svc);
      ul.appendChild(li);
    });
  }

  function renderKeysBlocks(container, feature) {
    container.innerHTML = '';
    if (!feature) return;
    var keysArr = feature.keys || [];
    keysArr.forEach(function(k) {
      var keyFrag = renderFromTemplate('keyRowTemplate', function(node){
        var root = node.firstElementChild || node;
        if (!root) return;
        var nameEl = root.querySelector('.feature-card__key-name');
        if (nameEl) nameEl.textContent = k.name || k.id || '';
        var paramsWrap = root.querySelector('.feature-card__params');
        if (paramsWrap) {
          var paramsArr = k.params || [];
          paramsArr.forEach(function(p){
            var pFrag = renderFromTemplate('paramRowTemplate', function(pn){
              var prow = pn.firstElementChild || pn;
              if (!prow) return;
              var pnEl = prow.querySelector('.feature-card__param-name');
              if (pnEl) pnEl.textContent = (p.name || p.id || '') + ':';
              var pvEl = prow.querySelector('.feature-card__param-value');
              if (pvEl) pvEl.textContent = (typeof p.value !== 'undefined' ? String(p.value) + '%' : '0%');
            });
            if (pFrag) paramsWrap.appendChild(pFrag);
          });
        }
      });
      if (keyFrag) container.appendChild(keyFrag);
    });
  }

  function applyActiveUi(article, isActive, isDeprecated) {
    var badge = article.querySelector('.feature-card__badge');
    var deprecatedBadge = article.querySelector('.feature-card__badge.deprecated');


    if (badge) {
      badge.setAttribute('data-badge', isActive ? 'active' : 'inactive');
      badge.textContent = isActive ? 'Используется' : 'Не используется';
    }

    if (isDeprecated) {
      article.classList.add('feature-card--deprecated');
      deprecatedBadge.textContent = 'Устарело';
    }
  }

  function needsAttention(f) { return !!f.is_deprecated; }

  function filterFeatures() {
    var result = features.slice();
    if (currentStatus === 'active') {
      result = result.filter(function(f){ return !!f.used; });
    } else if (currentStatus === 'inactive') {
      result = result.filter(function(f){ return !f.used; });
    } else if (currentStatus === 'attention') {
      result = result.filter(function(f){ return needsAttention(f); });
    }

    if (currentQuery) {
      var q = currentQuery.toLowerCase();
      result = result.filter(function(f){
        var name = (f.name || '').toLowerCase();
        var desc = (f.description || '').toLowerCase();
        return name.indexOf(q) !== -1 || desc.indexOf(q) !== -1;
      });
    }

    if (currentService) {
      result = result.filter(function(f){
        if (!Array.isArray(f.services)) return false;
        for (var i = 0; i < f.services.length; i++) {
          var s = f.services[i];
          if (!s) continue;
          // Prefer id match; fallback to name for legacy data
          if (s.id === currentService || (s.name && s.name === currentService)) return true;
        }
        return false;
      });
    }

    return result;
  }

  function renderList(data) {
    // Preserve loader element when re-rendering
    var loaderNode = null;
    if (listEl) {
      loaderNode = listEl.querySelector('#featuresLoader');
    }
    listEl.innerHTML = '';
    if (loaderNode) listEl.appendChild(loaderNode);
    // Server-driven pagination: render provided page items as-is
    var pageItems = data;

    pageItems.forEach(function(f) {
      var node = document.importNode(template.content, true);
      var article = node.querySelector('.feature-card');
      article.dataset.featureId = f.id;

      var titleEl = node.querySelector('.feature-card__title');
      var descEl = node.querySelector('.feature-card__description');
      var servicesEl = node.querySelector('.feature-card__services');
      var rangeEl = node.querySelector('.feature-card__range');
      var valueEl = node.querySelector('.feature-card__value');
      var keysEl = node.querySelector('.feature-card__keys');
      var deprecatedUpdatedEl = node.querySelector('.feature-card__deprecated-updated');
      var deleteBtn = node.querySelector('button[data-action="delete"]');

      if (titleEl) titleEl.textContent = f.name;
      if (descEl) descEl.textContent = f.description || '';
      applyActiveUi(article, !!f.used, !!f.is_deprecated);
      if (servicesEl) renderServices(servicesEl, f.services);
      if (deprecatedUpdatedEl) {
        var iso = f.updatedAt || f.updated_at || f.updated || new Date().toISOString();
        deprecatedUpdatedEl.textContent = formatDate(iso);
        deprecatedUpdatedEl.setAttribute('datetime', iso);
      }
      if (rangeEl) {
        var minVal = 100;
        var maxVal = 0;
        // own value
        if (typeof f.value === 'number') {
          minVal = Math.min(minVal, f.value);
          maxVal = Math.max(maxVal, f.value);
        }
        // keys' value and params' value
        var keysArr = Array.isArray(f.keys) ? f.keys : [];
        keysArr.forEach(function(k){
          var paramsArr = Array.isArray(k.params) ? k.params : [];
          paramsArr.forEach(function(p){
            if (typeof p.value === 'number') {
              minVal = Math.min(minVal, p.value);
              maxVal = Math.max(maxVal, p.value);
            }
          });
        });
        if (minVal === maxVal) {
          rangeEl.textContent = 'Активация: ' + String(minVal) + '%';
        } else {
          rangeEl.textContent = 'Активация: ' + String(minVal) + '% - ' + String(maxVal) + '%';
        }
      }
      if (valueEl) valueEl.textContent = 'Базовое распределение: ' + String(typeof f.value !== 'undefined' ? f.value : 0) + '%';
      if (keysEl) renderKeysBlocks(keysEl, f);
      if (deleteBtn) {
        deleteBtn.disabled = !!f.used;
        deleteBtn.title = f.used ? 'Нельзя удалить: фича используется' : '';
      }

      listEl.appendChild(node);
    });

    // Empty state depends on filtered result
    if (!emptyEl) return;
    emptyEl.hidden = data.length !== 0;

    // Update pagination UI based on server-reported total pages
    if (paginationEl && pageInfoEl) {
      var prevBtn = paginationEl.querySelector('button[data-page="prev"]');
      var nextBtn = paginationEl.querySelector('button[data-page="next"]');
      if (prevBtn) prevBtn.disabled = currentPage <= 1;
      if (nextBtn) nextBtn.disabled = currentPage >= serverTotalPages;
      pageInfoEl.textContent = 'Страница ' + currentPage + ' из ' + serverTotalPages;
      paginationEl.hidden = serverTotalPages <= 1;
    }
  }

  function render() {
    var data = filterFeatures();
    renderList(data);
  }

  function normalizeFeature(item) {
    if (!item || typeof item !== 'object') return null;
    var id = item.id != null ? String(item.id) : '';
    var name = item.name != null ? String(item.name) : id;
    var description = item.description != null ? String(item.description) : '';
    var value = typeof item.value === 'number' ? item.value : 0;
    var used = !!item.used;
    var isDeprecated = !!(item.is_deprecated);
    var createdAt = item.created_at || item.createdAt || new Date().toISOString();
    var updatedAt = item.updated_at || item.updatedAt || new Date().toISOString();
    var services = Array.isArray(item.services) ? item.services.map(function(s){
      return { id: s && s.id != null ? String(s.id) : '', name: s && s.name != null ? String(s.name) : '' };
    }) : [];
    var keys = Array.isArray(item.keys) ? item.keys.map(function(k){
      return {
        id: k && k.id != null ? String(k.id) : '',
        name: k && k.name != null ? String(k.name) : '',
        value: typeof (k && k.value) === 'number' ? k.value : 0,
        params: Array.isArray(k && k.params) ? k.params.map(function(p){
          return {
            id: p && p.id != null ? String(p.id) : '',
            name: p && p.name != null ? String(p.name) : '',
            value: typeof (p && p.value) === 'number' ? p.value : 0
          };
        }) : []
      };
    }) : [];
    return { id: id, name: name, description: description, value: value, used: used, is_deprecated: isDeprecated, services: services, keys: keys, createdAt: createdAt, updatedAt: updatedAt };
  }

  function buildFeaturesQuery() {
    var params = new URLSearchParams();
    params.set('page', String(currentPage));
    if (currentQuery) params.set('find', currentQuery);
    if (currentService) params.set('service_id', currentService);
    if (currentStatus === 'attention') params.set('is_deprecated', 'true');
    return params.toString();
  }

  function fetchFeatures() {
    try {
      var qs = buildFeaturesQuery();
      // show loader in features block only
      try {
        if (featuresLoaderEl) {
          featuresLoaderEl.setAttribute('aria-hidden', 'false');
          featuresLoaderEl.setAttribute('aria-busy', 'true');
        }
      } catch (_) {}
      // Abort previous request if still in-flight
      try {
        if (featuresFetchController && typeof featuresFetchController.abort === 'function') {
          featuresFetchController.abort();
        }
      } catch (_) {}
      featuresFetchController = (typeof AbortController !== 'undefined') ? new AbortController() : null;
      var requestSeq = ++featuresFetchSeq;
      var signal = featuresFetchController && featuresFetchController.signal ? { signal: featuresFetchController.signal } : {};
      api.get('/api/features' + (qs ? ('?' + qs) : ''), signal)
        .then(function(body){
          // Ignore stale responses from aborted/older requests
          if (requestSeq !== featuresFetchSeq) return;
          var items = Array.isArray(body && body.features) ? body.features : [];
          var norm = [];
          for (var i = 0; i < items.length; i++) {
            var nf = normalizeFeature(items[i]);
            if (nf) norm.push(nf);
          }
          // Sort by createdAt desc (fallback to updatedAt)
          try {
            norm.sort(function(a, b){
              var bd = Date.parse(b.createdAt || b.updatedAt);
              var ad = Date.parse(a.createdAt || a.updatedAt);
              if (isNaN(bd)) bd = 0;
              if (isNaN(ad)) ad = 0;
              return bd - ad;
            });
          } catch (_) {}
          features = norm;
          serverTotalPages = Math.max(1, parseInt(body && body.total_pages, 10) || 1);
          var pg = parseInt(body && body.page, 10);
          if (!isNaN(pg) && pg > 0) currentPage = pg;
          saveUiState();
          render();
        })
        .catch(function(){ /* ignore */ })
        .finally(function(){
          // Only hide loader if this is the latest request completing
          if (requestSeq === featuresFetchSeq) {
            try {
              if (featuresLoaderEl) {
                featuresLoaderEl.setAttribute('aria-busy', 'false');
                featuresLoaderEl.setAttribute('aria-hidden', 'true');
              }
            } catch (_) {}
            // Clear controller reference
            featuresFetchController = null;
          }
        });
    } catch (e) { /* no-op */ }
  }

  function findIndexById(id) {
    for (var i = 0; i < features.length; i++) {
      if (features[i].id === id) return i;
    }
    return -1;
  }

  function uid() {
    return 'f_' + Math.random().toString(36).slice(2, 9);
  }

  // ===== Modal helpers =====
  function openUiModal(title, renderContent) {
    var modal = document.getElementById('modal');
    if (!modal) return;
    var content = modal.querySelector('.modal__content');
    if (!content) return;
    content.innerHTML = '';
    var wrap = document.createElement('div');
    wrap.className = 'modal-form';
    content.appendChild(wrap);

    function close() {
      modal.setAttribute('aria-hidden', 'true');
      document.body.style.overflow = '';
    }

    renderContent(wrap, close);

    modal.setAttribute('aria-hidden', 'false');
    document.body.style.overflow = 'hidden';
  }

  // ===== Services modal =====
  function openServicesModal(featureIndex) {
    var original = features[featureIndex];
    var draft = JSON.parse(JSON.stringify({ services: original.services || [] }));
    var featureId = original && original.id ? String(original.id) : '';

    openUiModal('Сервисы фичи: ' + (original.name || ''), function(root, close){
      var tpl = document.getElementById('servicesEditTemplate');
      if (!tpl) return;
      var node = document.importNode(tpl.content, true);
      root.appendChild(node);
      var titleEl = root.querySelector('.modal__title');
      if (titleEl) titleEl.textContent = 'Сервисы фичи: ' + (original.name || '');

      var listEl = root.querySelector('#svcList');
      var nameInput = root.querySelector('#svcName');
      var addBtn = root.querySelector('#svcAdd');

      // Replace free-text input with dropdown sourced from loaded services
      var selectEl = document.createElement('select');
      selectEl.id = 'svcSelect';
      selectEl.className = 'features__service-select';
      var chooseOpt = document.createElement('option');
      chooseOpt.value = '';
      chooseOpt.textContent = 'Выберите сервис';
      selectEl.appendChild(chooseOpt);
      var catalog = Array.isArray(window.__servicesCatalog) ? window.__servicesCatalog : [];

      function rebuildSelectOptions() {
        // Preserve current selection if still available
        var current = selectEl.value || '';
        // Clear all options, re-add placeholder
        selectEl.innerHTML = '';
        selectEl.appendChild(chooseOpt.cloneNode(true));

        var used = {};
        (draft.services || []).forEach(function(s){
          var id = s && s.id != null ? String(s.id) : '';
          if (id) used[id] = true;
        });

        var list = Array.isArray(window.__servicesCatalog) ? window.__servicesCatalog : catalog;
        list.forEach(function(s){
          var id = s && s.id != null ? String(s.id) : '';
          if (!id || used[id]) return; // skip already used
          var opt = document.createElement('option');
          opt.value = id;
          opt.textContent = (s && s.name) ? s.name : id;
          selectEl.appendChild(opt);
        });

        // restore selection if still valid
        if (current && !used[current] && Array.prototype.some.call(selectEl.options, function(o){ return o.value === current; })) {
          selectEl.value = current;
        } else {
          selectEl.value = '';
        }
      }
      if (nameInput && nameInput.parentNode) {
        nameInput.parentNode.replaceChild(selectEl, nameInput);
      }

      function renderList() {
        listEl.innerHTML = '';
        (draft.services || []).forEach(function(s){
          var li = document.createElement('li');
          var label = (s && (s.name || s.id)) || '';
          li.innerHTML = '<span>' + label + '</span> ' +
            '<button type="button" class="btn btn--danger" data-remove="' + label.replace(/"/g, '&quot;') + '">Удалить</button>';
          if (s && s.id != null) {
            li.setAttribute('data-service-id', String(s.id));
          }
          listEl.appendChild(li);
        });
        rebuildSelectOptions();
      }

      root.addEventListener('click', function(e){
        var btn = e.target.closest('button');
        if (!btn) return;
        var remove = btn.getAttribute('data-remove');
        if (remove) {
          var li = btn.closest('li');
          var sid = li ? String(li.getAttribute('data-service-id') || '') : '';
          if (!sid || !featureId) return;

          var ok = true;
          try { ok = window.confirm('Удалить сервис из фичи?'); } catch (_) {}
          if (!ok) return;

          var originalHtml = btn.innerHTML;
          btn.innerHTML = 'Удаляем…';
          btn.disabled = true;
          btn.setAttribute('aria-busy', 'true');

          api.del('/api/features/' + encodeURIComponent(featureId) + '/services/' + encodeURIComponent(sid))
          .then(function(){
            draft.services = (draft.services || []).filter(function(s){ return String(s && s.id) !== sid; });
            // sync main list in background
            try { fetchFeatures(); } catch (_) {}
            renderList();
          })
          .catch(function(){
            try { window.alert('Не удалось удалить сервис из фичи. Повторите попытку.'); } catch (_) {}
            if (btn && btn.isConnected) {
              btn.innerHTML = originalHtml;
              btn.disabled = false;
              btn.removeAttribute('aria-busy');
            }
          });
        }
      });

      addBtn.addEventListener('click', function(){
        var id = String(selectEl.value || '').trim();
        if (!id || !featureId) return;
        if (!draft.services) draft.services = [];
        if (!draft.services.some(function(s){ return s && String(s.id) === id; })) {
          var found = null;
          for (var i = 0; i < catalog.length; i++) {
            if (catalog[i] && catalog[i].id === id) { found = catalog[i]; break; }
          }
          var displayName = (found && found.name) || id;
          var originalHtml = addBtn.innerHTML;
          addBtn.innerHTML = 'Добавляем…';
          addBtn.disabled = true;
          addBtn.setAttribute('aria-busy', 'true');

          api.post('/api/features/' + encodeURIComponent(featureId) + '/services/' + encodeURIComponent(id), {})
          .then(function(){
            draft.services.push({ id: id, name: displayName });
            selectEl.value = '';
            // sync main list in background
            try { fetchFeatures(); } catch (_) {}
            renderList();
          })
          .catch(function(){
            try { window.alert('Не удалось добавить сервис к фиче. Повторите попытку.'); } catch (_) {}
          })
          .finally(function(){
            addBtn.innerHTML = originalHtml;
            addBtn.disabled = false;
            addBtn.removeAttribute('aria-busy');
          });
        }
      });

      renderList();
    });
  }

  // ===== Attributes modal =====
  function openAttributesModal(featureIndex) {
    var original = features[featureIndex];
    var draft = JSON.parse(JSON.stringify({ value: typeof original.value === 'number' ? original.value : 0, keys: original.keys || [] }));
    var featureId = original && original.id ? String(original.id) : '';

    function renderKeys(root) {
      var wrap = root.querySelector('#keysWrap');
      wrap.innerHTML = '';
      (draft.keys || []).forEach(function(k, ki){
        var block = document.createElement('div');
        block.className = 'key-block';
        var html = '' +
          '<div class="key-head">' +
            '<div class="key-title">' + (k.name || k.id || '') + '</div>' +
            '<div>' +
              '<button type="button" class="btn btn--ghost btn--sm remove-key" data-ki="' + ki + '">Удалить ключ</button>' +
            '</div>' +
          '</div>' +
          '<div class="params" data-ki="' + ki + '">';
        (k.params || []).forEach(function(p, pi){
          html += '' +
            '<div class="row row--3" data-pi="' + pi + '">' +
              '<label style="display:contents">' +
                '<span class="feature-card__param-name">' + (p.name || p.id || '') + ':</span>' +
                '<input type="number" min="0" max="100" class="param-input" value="' + (typeof p.value === 'number' ? p.value : 0) + '" />' +
              '</label>' +
              '<button type="button" class="btn btn--danger btn--sm remove-param" data-ki="' + ki + '" data-pi="' + pi + '">X</button>' +
            '</div>';
        });
        html += '</div>' +
          '<div class="row add-param-row" data-ki="' + ki + '">' +
            '<input type="text" class="param-name-input" placeholder="Имя параметра" />' +
            '<input type="number" class="param-value-input" placeholder="%" min="0" max="100" />' +
            '<button type="button" class="btn btn--success add-param">Добавить параметр</button>' +
          '</div>';
        block.innerHTML = html;
        wrap.appendChild(block);
      });
    }

    openUiModal('Атрибуты фичи: ' + (original.name || ''), function(root, close){
      var tpl = document.getElementById('attributesEditTemplate');
      if (!tpl) return;
      var node = document.importNode(tpl.content, true);
      root.appendChild(node);
      var titleEl = root.querySelector('.modal__title');
      if (titleEl) titleEl.textContent = 'Атрибуты фичи: ' + (original.name || '');

      var baseInput = root.querySelector('#baseValue');
      baseInput.value = String(draft.value || 0);

      renderKeys(root);

      root.addEventListener('input', function(e){
        var input = e.target;
        if (input.id === 'baseValue') {
          var v = parseInt(input.value, 10);
          if (!isNaN(v)) draft.value = Math.max(0, Math.min(100, v));
          return;
        }
        if (input.classList.contains('param-input')) {
          var label = input.closest('label[row]');
        }
      });

      root.addEventListener('change', function(e){
        var input = e.target;
        if (input.classList.contains('param-input')) {
          var params = input.closest('.params');
          if (!params) return;
          var ki = parseInt(params.getAttribute('data-ki'), 10);
          var row = input.closest('.row');
          var pi = row ? parseInt(row.getAttribute('data-pi'), 10) : -1;
          var v = parseInt(input.value, 10);
          if (!isNaN(v) && draft.keys[ki] && draft.keys[ki].params && draft.keys[ki].params[pi]) {
            draft.keys[ki].params[pi].value = Math.max(0, Math.min(100, v));
          }
        }
      });

      root.addEventListener('click', function(e){
        var btn = e.target.closest('button');
        if (!btn) return;
        if (btn.classList.contains('remove-key')) {
          var rki = parseInt(btn.getAttribute('data-ki'), 10);
          if (!isNaN(rki)) {
            draft.keys.splice(rki, 1);
            renderKeys(root);
          }
          return;
        }
        if (btn.classList.contains('remove-param')) {
          var rki2 = parseInt(btn.getAttribute('data-ki'), 10);
          var rpi2 = parseInt(btn.getAttribute('data-pi'), 10);
          if (!isNaN(rki2) && !isNaN(rpi2) && draft.keys[rki2] && draft.keys[rki2].params) {
            draft.keys[rki2].params.splice(rpi2, 1);
            renderKeys(root);
          }
          return;
        }
        if (btn.id === 'addKeyBtn') {
          var nameEl = root.querySelector('#newKeyName');
          var name = String(nameEl.value || '').trim();
          if (!name) return;
          if (!draft.keys) draft.keys = [];
          draft.keys.push({ id: 'k_' + Math.random().toString(36).slice(2, 9), name: name, params: [] });
          nameEl.value = '';
          renderKeys(root);
          return;
        }
        if (btn.classList.contains('add-param')) {
          var row = btn.closest('.add-param-row');
          var ki = parseInt(row.getAttribute('data-ki'), 10);
          var pn = String((row.querySelector('.param-name-input').value || '').trim());
          var pv = parseInt(row.querySelector('.param-value-input').value, 10);
          if (!pn) return;
          if (isNaN(pv)) pv = 0;
          if (!draft.keys[ki].params) draft.keys[ki].params = [];
          draft.keys[ki].params.push({ id: 'p_' + Math.random().toString(36).slice(2, 9), name: pn, value: Math.max(0, Math.min(100, pv)) });
          row.querySelector('.param-name-input').value = '';
          row.querySelector('.param-value-input').value = '';
          renderKeys(root);
          return;
        }
      });

      var saveBtn = root.querySelector('#attrSave');
      var isSubmitting = false;
      var originalBtnHtml = null;
      var progress = {
        featureUpdated: false,
        createdKeyIdByTemp: {}, // tempKeyId -> realKeyId
        createdParamIdByTemp: {}, // tempParamId -> realParamId
        deletedKeyIds: {}, // realKeyId -> true
        deletedParamIds: {} // realParamId -> true
      };
      var overlay = root.querySelector('.modal__saving-overlay');

      function setLoading(on) {
        if (!saveBtn) return;
        if (on) {
          isSubmitting = true;
          originalBtnHtml = saveBtn.innerHTML;
          saveBtn.innerHTML = 'Сохраняем…';
          saveBtn.disabled = true;
          saveBtn.setAttribute('aria-busy', 'true');
          if (overlay) overlay.setAttribute('aria-hidden', 'false');
        } else {
          isSubmitting = false;
          saveBtn.disabled = false;
          saveBtn.removeAttribute('aria-busy');
          if (originalBtnHtml != null) saveBtn.innerHTML = originalBtnHtml;
          if (overlay) overlay.setAttribute('aria-hidden', 'true');
        }
      }

      function runSequentially(tasks) {
        return tasks.reduce(function(p, task){
          return p.then(task);
        }, Promise.resolve());
      }

      function isTempId(id, prefix) {
        return typeof id === 'string' && id.indexOf(prefix) === 0;
      }

      saveBtn.addEventListener('click', function(){
        if (isSubmitting) return;
        if (!featureId) { try { window.alert('Неизвестный id фичи'); } catch (_) {} return; }

        // Build diffs
        var tasks = [];

        // 1) Feature base value update
        var originalValue = (typeof original.value === 'number' ? original.value : 0);
        var draftValue = (typeof draft.value === 'number' ? Math.max(0, Math.min(100, draft.value)) : 0);
        if (draftValue !== originalValue && !progress.featureUpdated) {
          tasks.push(function(){
            return api.put('/api/features/' + encodeURIComponent(featureId), { name: original.name || '', description: original.description || '', value: draftValue })
              .then(function(){ progress.featureUpdated = true; });
          });
        }

        // Index originals
        var originalKeysById = {};
        (Array.isArray(original.keys) ? original.keys : []).forEach(function(k){ originalKeysById[String(k.id)] = k; });
        var originalKeyIds = Object.keys(originalKeysById);

        // Draft keys
        var draftKeys = Array.isArray(draft.keys) ? draft.keys : [];
        var draftKeysById = {};
        draftKeys.forEach(function(k){ draftKeysById[String(k.id)] = k; });

        var removedKeyIds = originalKeyIds.filter(function(id){ return !draftKeysById.hasOwnProperty(id); });
        var newKeys = draftKeys.filter(function(k){ return isTempId(k.id, 'k_'); });
        var existingDraftKeys = draftKeys.filter(function(k){ return !isTempId(k.id, 'k_'); });

        // Helper to create single param, idempotent with temp id tracking
        function createParamTask(keyId, paramObj) {
          return function(){
            var pid = String(paramObj.id);
            if (isTempId(pid, 'p_')) {
              // If already created previously, just materialize id and skip network
              if (progress.createdParamIdByTemp[pid]) {
                paramObj.id = progress.createdParamIdByTemp[pid];
                return Promise.resolve();
              }
              return api.post('/api/keys/' + encodeURIComponent(keyId) + '/params', { feature_id: featureId, name: paramObj.name || paramObj.id || '', value: (typeof paramObj.value === 'number' ? Math.max(0, Math.min(100, paramObj.value)) : 0) })
              .then(function(body){ body = body || {}; return body; })
              .then(function(body){
                var realPid = body && body.id ? String(body.id) : '';
                if (!realPid) throw new Error('param_create_no_id');
                progress.createdParamIdByTemp[pid] = realPid;
                paramObj.id = realPid; // materialize to avoid duplicate on retry
              });
            }
            return Promise.resolve();
          };
        }

        // 2) Create new keys (and their params afterwards), idempotent by temp id
        newKeys.forEach(function(nk){
          tasks.push(function(){
            var tempId = String(nk.id);
            if (progress.createdKeyIdByTemp[tempId]) {
              // Already created previously, materialize id and only create any remaining params
              var realKeyIdPrev = progress.createdKeyIdByTemp[tempId];
              nk.id = realKeyIdPrev;
              var paramsPrev = Array.isArray(nk.params) ? nk.params : [];
              return runSequentially(paramsPrev.map(function(p){ return createParamTask(realKeyIdPrev, p); }));
            }
            return api.post('/api/features/' + encodeURIComponent(featureId) + '/keys', { key: nk.name || nk.id || '', description: '', value: 0 })
            .then(function(body){ body = body || {}; return body; })
            .then(function(body){
              var realId = body && body.id ? String(body.id) : '';
              if (!realId) throw new Error('key_create_no_id');
              progress.createdKeyIdByTemp[tempId] = realId;
              nk.id = realId; // materialize to avoid duplicate on retry
              // Create params under this new key
              var params = Array.isArray(nk.params) ? nk.params : [];
              return runSequentially(params.map(function(p){ return createParamTask(realId, p); }));
            });
          });
        });

        // 3) For existing keys, process param diffs
        existingDraftKeys.forEach(function(ek){
          var keyId = String(ek.id);
          var oKey = originalKeysById[keyId] || { params: [] };
          var oParamsById = {};
          (Array.isArray(oKey.params) ? oKey.params : []).forEach(function(p){ oParamsById[String(p.id)] = p; });
          var oParamIds = Object.keys(oParamsById);

          var dParams = Array.isArray(ek.params) ? ek.params : [];
          var dParamsById = {};
          dParams.forEach(function(p){ dParamsById[String(p.id)] = p; });

          var removedParamIds = oParamIds.filter(function(id){ return !dParamsById.hasOwnProperty(id); });
          var newParams = dParams.filter(function(p){ return isTempId(p.id, 'p_'); });
          var commonParamIds = oParamIds.filter(function(id){ return dParamsById.hasOwnProperty(id); });
          var updatedParams = commonParamIds.filter(function(id){
            var op = oParamsById[id];
            var dp = dParamsById[id];
            var ov = (typeof op.value === 'number' ? op.value : 0);
            var dv = (typeof dp.value === 'number' ? Math.max(0, Math.min(100, dp.value)) : 0);
            var on = op.name || '';
            var dn = dp.name || '';
            return (ov !== dv) || (on !== dn);
          });

          // New params
          newParams.forEach(function(np){
            tasks.push(createParamTask(keyId, np));
          });

          // Updated params
          updatedParams.forEach(function(pid){
            var dp = dParamsById[pid];
            tasks.push(function(){
            return api.put('/api/params/' + encodeURIComponent(pid), { feature_id: featureId, key_id: keyId, name: dp.name || '', value: (typeof dp.value === 'number' ? Math.max(0, Math.min(100, dp.value)) : 0) });
            });
          });

          // Removed params
          removedParamIds.forEach(function(pid){
            if (progress.deletedParamIds[pid]) return;
            tasks.push(function(){
            return api.del('/api/params/' + encodeURIComponent(pid)).then(function(){ progress.deletedParamIds[pid] = true; });
            });
          });
        });

        // 4) Removed keys
        removedKeyIds.forEach(function(kid){
          if (progress.deletedKeyIds[kid]) return;
          tasks.push(function(){
            return api.del('/api/keys/' + encodeURIComponent(kid)).then(function(){ progress.deletedKeyIds[kid] = true; });
          });
        });

        setLoading(true);
        runSequentially(tasks)
          .then(function(){
            // Refresh list from server
            fetchFeatures();
            close();
          })
          .catch(function(){
            try { window.alert('Не удалось сохранить атрибуты. Повторите попытку.'); } catch (_) {}
          })
          .finally(function(){
            setLoading(false);
          });
      });
    });
  }

  function openCreateFeatureModal() {
    openUiModal('Новая фича', function(root, close){
      var tpl = document.getElementById('featureCreateTemplate');
      if (!tpl) return;
      var node = document.importNode(tpl.content, true);
      root.appendChild(node);
      var nameEl = root.querySelector('#createFeatureName');
      var descEl = root.querySelector('#createFeatureDesc');
      var saveBtn = root.querySelector('#createFeatureSave');
      var isSubmitting = false;
      var originalBtnHtml = null;

      function setLoading(on) {
        if (!saveBtn) return;
        if (on) {
          isSubmitting = true;
          originalBtnHtml = saveBtn.innerHTML;
          saveBtn.innerHTML = 'Создаем…';
          saveBtn.disabled = true;
          saveBtn.setAttribute('aria-busy', 'true');
        } else {
          isSubmitting = false;
          saveBtn.disabled = false;
          saveBtn.removeAttribute('aria-busy');
          if (originalBtnHtml != null) saveBtn.innerHTML = originalBtnHtml;
        }
      }
      if (saveBtn) {
        saveBtn.addEventListener('click', function(){
          if (isSubmitting) return;
          var name = String((nameEl && nameEl.value) || '').trim();
          var description = String((descEl && descEl.value) || '').trim();
          if (!name) { if (nameEl) nameEl.focus(); return; }

          setLoading(true);
          api.post('/api/features', { name: name, description: description })
          .then(function(){
            fetchFeatures();
            close();
          })
          .catch(function(){
            try { window.alert('Не удалось создать фичу. Повторите попытку.'); } catch (_) {}
          })
          .finally(function(){
            setLoading(false);
          });
        });
      }
    });
  }

  function openCreateServiceModal() {
    openUiModal('Новый сервис', function(root, close){
      var tpl = document.getElementById('serviceCreateTemplate');
      if (!tpl) return;
      var node = document.importNode(tpl.content, true);
      root.appendChild(node);

      var nameEl = root.querySelector('#createServiceName');
      var saveBtn = root.querySelector('#createServiceSave');
      var isSubmitting = false;
      var originalBtnHtml = null;

      function setLoading(on) {
        if (!saveBtn) return;
        if (on) {
          isSubmitting = true;
          originalBtnHtml = saveBtn.innerHTML;
          saveBtn.innerHTML = 'Создаем…';
          saveBtn.disabled = true;
          saveBtn.setAttribute('aria-busy', 'true');
        } else {
          isSubmitting = false;
          saveBtn.disabled = false;
          saveBtn.removeAttribute('aria-busy');
          if (originalBtnHtml != null) saveBtn.innerHTML = originalBtnHtml;
        }
      }

      function normalizeServiceLocal(item) {
        var id = item && (item.id != null ? String(item.id) : '');
        var name = item && (item.name != null ? String(item.name) : id);
        var active = !!(item && item.active);
        return { id: id, name: name, active: active };
      }

      if (saveBtn) {
        saveBtn.addEventListener('click', function(){
          if (isSubmitting) return;
          var name = String((nameEl && nameEl.value) || '').trim();
          if (!name) { if (nameEl) nameEl.focus(); return; }

          setLoading(true);
          api.post('/api/services', { name: name })
          .then(function(){ return api.get('/api/services'); })
          .then(function(arr){
            var norm = Array.isArray(arr) ? arr.map(normalizeServiceLocal) : [];
            window.__servicesCatalog = norm;
            if (typeof window.__refreshServicesUi === 'function') {
              window.__refreshServicesUi();
            }
            close();
          })
          .catch(function(){
            try { window.alert('Не удалось создать сервис. Повторите попытку.'); } catch (_) {}
          })
          .finally(function(){
            setLoading(false);
          });
        });
      }
    });
  }

  function onListClick(e) {
    var btn = e.target.closest('button');
    if (!btn) return;
    var article = e.target.closest('.feature-card');
    if (!article) return;
    var id = article.dataset.featureId;
    var index = findIndexById(id);
    if (index === -1) return;

    var action = btn.getAttribute('data-action');
    if (action === 'toggle-active') {
      features[index].active = !features[index].active;
      features[index].updatedAt = nowIso();
      render();
    } else if (action === 'delete') {
      if (features[index].used) {
        return; // deletion disabled for used features
      }
      var ok = true;
      try { ok = window.confirm('Удалить фичу «' + (features[index].name || '') + '»?'); } catch (_) {}
      if (!ok) return;

      // show button loading state
      var originalHtml = btn.innerHTML;
      btn.innerHTML = 'Удаляем…';
      btn.disabled = true;
      btn.setAttribute('aria-busy', 'true');

      api.del('/api/features/' + encodeURIComponent(id))
        .then(function(){ fetchFeatures(); })
        .catch(function(){
          try { window.alert('Не удалось удалить фичу. Повторите попытку.'); } catch (_) {}
          // restore button state if still in DOM (list may re-render)
          if (btn && btn.isConnected) {
            btn.innerHTML = originalHtml;
            btn.disabled = false;
            btn.removeAttribute('aria-busy');
          }
        });
    } else if (action === 'edit') {
      openServicesModal(index);
    } else if (action === 'edit_attr') {
      openAttributesModal(index);
    }
  }

  // legacy createFeatureFlow removed (replaced by openCreateFeatureModal)

  // Remove previously attached modal-opening listener by cloning the button
  if (addBtn && addBtn.parentNode) {
    var clone = addBtn.cloneNode(true);
    addBtn.parentNode.replaceChild(clone, addBtn);
    clone.addEventListener('click', function() {
      openCreateFeatureModal();
    });
  }

  // Wire "Добавить сервис" button in services overlay header to open service create modal
  var addServiceBtn = document.getElementById('openServiceModalBtn');
  if (addServiceBtn && addServiceBtn.parentNode) {
    var addServiceClone = addServiceBtn.cloneNode(true);
    addServiceBtn.parentNode.replaceChild(addServiceClone, addServiceBtn);
    addServiceClone.addEventListener('click', function(e) {
      e.preventDefault();
      e.stopPropagation();
      openCreateServiceModal();
    });
  }

  listEl.addEventListener('click', onListClick);
  // Populate service filter options from loaded catalog
  function populateServiceOptions() {
    if (!serviceSelectEl) return;
    var prev = serviceSelectEl.value;
    var catalog = Array.isArray(window.__servicesCatalog) ? window.__servicesCatalog : [];
    serviceSelectEl.innerHTML = '';
    var allOpt = document.createElement('option');
    allOpt.value = '';
    allOpt.textContent = 'Все сервисы';
    serviceSelectEl.appendChild(allOpt);
    catalog.forEach(function(s){
      var opt = document.createElement('option');
      opt.value = s.id;
      opt.textContent = s.name || s.id;
      serviceSelectEl.appendChild(opt);
    });
    // Prefer saved currentService; fallback to previous DOM value
    var desired = currentService || prev;
    if (desired && catalog.some(function(s){ return s.id === desired; })) {
      serviceSelectEl.value = desired;
      currentService = desired;
    } else {
      serviceSelectEl.value = '';
      currentService = '';
    }
  }

  // Expose for services loader
  window.__populateServiceOptions = populateServiceOptions;

  function setPressedInGroup(groupEl, buttonEl) {
    if (!groupEl || !buttonEl) return;
    var buttons = groupEl.querySelectorAll('button');
    buttons.forEach(function(b){ b.setAttribute('aria-pressed', 'false'); });
    buttonEl.setAttribute('aria-pressed', 'true');
  }

  // Wire controls
  if (statusToggleEl) {
    statusToggleEl.addEventListener('click', function(e){
      var btn = e.target.closest('button[data-status]');
      if (!btn) return;
      currentStatus = btn.getAttribute('data-status') || 'all';
      setPressedInGroup(statusToggleEl, btn);
      currentPage = 1;
      saveUiState();
      fetchFeatures();
    });
  }

  if (viewToggleEl) {
    viewToggleEl.addEventListener('click', function(e){
      var btn = e.target.closest('button[data-view]');
      if (!btn) return;
      currentView = btn.getAttribute('data-view') || 'detailed';
      setPressedInGroup(viewToggleEl, btn);
      if (featuresSection) {
        if (currentView === 'simple') {
          featuresSection.classList.add('features--simple');
        } else {
          featuresSection.classList.remove('features--simple');
        }
      }
      saveUiState();
    });
  }

  if (searchInputEl) {
    searchInputEl.addEventListener('input', function(){
      currentQuery = String(searchInputEl.value || '').trim();
      saveUiState();
      currentPage = 1;
      fetchFeatures();
    });
  }

  if (serviceSelectEl) {
    serviceSelectEl.addEventListener('change', function(){
      currentService = serviceSelectEl.value || '';
      currentPage = 1;
      saveUiState();
      fetchFeatures();
    });
  }

  if (paginationEl) {
    paginationEl.addEventListener('click', function(e){
      var btn = e.target.closest('button[data-page]');
      if (!btn) return;
      var dir = btn.getAttribute('data-page');
      if (dir === 'prev' && currentPage > 1) {
        currentPage -= 1;
        saveUiState();
        fetchFeatures();
      } else if (dir === 'next') {
        currentPage += 1;
        saveUiState();
        fetchFeatures();
      }
    });
  }

  // Apply saved UI state before initial render
  applyInitialUiState();

  // Initial setup
  populateServiceOptions();
  // Ensure we persist normalized values after options are populated
  saveUiState();
  fetchFeatures();
})();
