// Minimal JS runtime version generated from TS client
(function (global) {
  function clampPercent(p) { if (p < 0) return 0; if (p > 100) return 100; return p; }
  function bucketHit(featureName, seed, percent) {
    if (percent <= 0) return false; if (percent >= 100) return true;
    var h = 0xcbf29ce484222325n, prime = 0x100000001b3n; var s = featureName + '::' + seed;
    for (var i = 0; i < s.length; i++) { h ^= BigInt(s.charCodeAt(i)); h = (h * prime) & 0xFFFFFFFFFFFFFFFFn; }
    var bucket = Number(h % 100n); return bucket < percent;
  }
  function detectStorage(prefix) {
    try { if (typeof window !== 'undefined' && window.localStorage) return {
      getItem: function(k){ return window.localStorage.getItem(prefix ? prefix+':'+k : k); },
      setItem: function(k,v){ window.localStorage.setItem(prefix ? prefix+':'+k : k, v); },
      removeItem: function(k){ window.localStorage.removeItem(prefix ? prefix+':'+k : k); }
    }; } catch(e) {}
    var mem = {};
    return { getItem: function(k){ return Object.prototype.hasOwnProperty.call(mem,k) ? mem[k] : null; }, setItem: function(k,v){ mem[k]=v; }, removeItem: function(k){ delete mem[k]; } };
  }
  function Client(baseUrl, serviceName, opts) {
    if (!baseUrl) throw new Error('baseUrl is required');
    if (!serviceName) throw new Error('serviceName is required');
    opts = opts || {};
    this.baseUrl = baseUrl.replace(/\/?$/, '');
    this.serviceName = serviceName;
    this.autoStats = opts.autoSendStats !== false;
    this.onUpdate = opts.onUpdate;
    this.lastVersion = opts.initialVersion || 0;
    this.statsFlushIntervalMs = Math.max(1000, opts.statsFlushIntervalMs || 180000);
    this.storage = opts.storage || detectStorage(opts.storagePrefix || 'fc');
    this.features = new Map();
    this.stats = new Set();
    this._restore();
    var self = this; this._timer = setInterval(function(){ self.flushStats(); }, this.statsFlushIntervalMs);
    // auto-start polling
    this._abort = new AbortController ? new AbortController() : null;
    this.pollOnce(this._abort ? this._abort.signal : undefined);
    var every = Math.max(500, opts.pollIntervalMs || 3000);
    this.startPolling(every, this._abort ? this._abort.signal : undefined);
  }
  Client.prototype._cacheKey = function(){ return 'fc:'+this.serviceName+':v1'; };
  Client.prototype._persist = function(){ try { var snap = this.getSnapshot(); this.storage.setItem(this._cacheKey(), JSON.stringify({version:this.lastVersion, features:snap})); } catch(e){} };
  Client.prototype._restore = function(){ try { var raw = this.storage.getItem(this._cacheKey()); if(!raw) return; var obj = JSON.parse(raw); this.lastVersion = obj.version||0; this.features.clear(); var f = obj.features||{}; Object.keys(f).forEach((k)=>{ var v = f[k]; this.features.set(k,{ name:v.name, all:+v.all, keys:v.keys||{} }); }); } catch(e){} };
  Client.prototype.applyUpdate = function(data){
    (data.features||[]).forEach((f)=>{
      var name = f.name;
      var existing = this.features.get(name) || { name:name, all:0, keys:{} };
      if (+f.all !== -1) existing.all = +f.all;
      if (!existing.keys) existing.keys = {};
      (f.props||[]).forEach((p)=>{
        var keyName = p.name;
        var kc = existing.keys[keyName] || { all:0, items:{} };
        if (+p.all !== -1) kc.all = +p.all;
        Object.keys(p.item||{}).forEach((k)=>{ kc.items[k] = +p.item[k]; });
        existing.keys[keyName] = kc;
      });
      this.features.set(name, existing);
    });
    (data.deleted||[]).forEach((d)=>{
      if (d.kind===0) this.features.delete(d.feature_name);
      else if (d.kind===1){ var feat=this.features.get(d.feature_name); if (feat && d.key_name) delete feat.keys[d.key_name]; }
      else if (d.kind===2){ var feat2=this.features.get(d.feature_name); if (feat2 && d.key_name && d.param_name) delete (feat2.keys[d.key_name]||{}).items[d.param_name]; }
    });
    if (data.version > this.lastVersion) this.lastVersion = data.version;
    this._persist();
  };
  Client.prototype.pollOnce = async function(signal){
    var resp = await fetch(this.baseUrl + '/api/updates', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ service_name:this.serviceName, last_version:this.lastVersion }), signal });
    if (!resp.ok) return null; var data = await resp.json(); this.applyUpdate(data);
    var changed = []; (data.features||[]).forEach((f)=>{ var v=this.features.get(f.name); if (v) changed.push(v); });
    var evt = { version: this.lastVersion, features: changed }; if (this.onUpdate && changed.length) this.onUpdate(evt); return evt;
  };
  Client.prototype.startPolling = async function(intervalMs, signal){ intervalMs = Math.max(500, intervalMs||3000); var wait = intervalMs; while(!signal || !signal.aborted){ try{ await this.pollOnce(signal); wait = intervalMs; } catch(e){ wait = Math.min(10000, Math.max(500, Math.floor(wait*1.5))); } await new Promise(function(r){ setTimeout(r, wait); }); } };
  Client.prototype.isEnabled = function(featureName, seed, attrs){ var cfg=this.features.get(featureName); if(!cfg) return false; var percent=-1, keyLevel=null; if(attrs){ for (var k in attrs){ var v=attrs[k]; var kc=cfg.keys[k]; if(!kc) continue; if (kc.items[v]!==undefined){ percent = kc.items[v]; break; } if (keyLevel===null) keyLevel = kc.all; } } if (percent<0 && keyLevel!==null){ percent=keyLevel; } if (percent<0){ percent=cfg.all; } percent=clampPercent(percent); var enabled=bucketHit(featureName, seed, percent); if (enabled && this.autoStats) this.track(featureName); return enabled; };
  Client.prototype.track = function(featureName){ if (featureName) this.stats.add(featureName); };
  Client.prototype.getSnapshot = function(){ var out={}; this.features.forEach(function(v,k){ var keys={}; Object.keys(v.keys).forEach(function(kk){ var kv=v.keys[kk]; keys[kk] = { all: kv.all, items: Object.assign({}, kv.items) }; }); out[k] = { name: v.name, all: v.all, keys: keys }; }); return out; };
  Client.prototype.flushStats = async function(signal){ if (this.stats.size===0) return; var list = Array.from(this.stats); this.stats.clear(); try { await fetch(this.baseUrl + '/api/stats', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ service_name:this.serviceName, features:list }), signal }); } catch(e){ list.forEach((f)=>this.stats.add(f)); } };
  Client.prototype.close = function(){ try{ if (this._abort) this._abort.abort(); } catch(e){} if (this._timer) { clearInterval(this._timer); this._timer = null; } };
  if (typeof module !== 'undefined' && module.exports) module.exports = { Client: Client };
  else global.FeatureChaosHttpClient = Client;
})(typeof window !== 'undefined' ? window : globalThis);
