let __state = { features: [], services: [] };

async function fetchFeatures(){
  const res = await fetch('/api/features');
  const data = await res.json();
  __state.features = Array.isArray(data) ? data : [];
  const tbody = document.getElementById('features');
  tbody.innerHTML='';
  const allServices = await fetchServices();
  __state.services = Array.isArray(allServices) ? allServices : [];
  renderServicesSidebar(allServices);
  const filter = (document.getElementById('f-search')?.value||'').toLowerCase();
  const list = (__state.features||[]).filter(f => !filter || (f.name||'').toLowerCase().includes(filter) || (f.description||'').toLowerCase().includes(filter));
  for(const f of list){
    const tr = document.createElement('tr');
    const keys = Array.isArray(f.keys) ? f.keys : [];
    const serviceIds = new Set((f.services||[]).map(s=>s.id));
    const availableServices = allServices.filter(s=>!serviceIds.has(s.id));
    const keysHtmlResolved = [];
    for(const k of keys){
      const params = Array.isArray(k.params) ? k.params : [];
      const paramsHtml = params.map(p=>
        '<div class="params">'
        + p.name + ' = '
        + '<input type="number" id="p-value-'+p.id+'" value="'+p.value+'" class="value-input mini" min="0" max="100"/> % '
        + '<button class="mini" onclick="setParamValue(\''+p.id+'\')">Save</button> '
        + '<button class="mini" onclick="deleteParam(\''+p.id+'\')">✖</button>'
        + '</div>'
      ).join('');
      keysHtmlResolved.push('<div class="keys">'
        + '<div class="k-header"><span class="k-title">'+(k.name||'')+'</span> = '
        + '<input type="number" id="k-value-'+k.id+'" value="'+(k.value||0)+'" class="value-input mini" min="0" max="100"/> % '
        + '<button class="mini" onclick="setKeyValue(\''+k.id+'\')">Save</button> '
        + '<button class="mini" onclick="deleteKey(\''+k.id+'\')">✖</button>'
        + '</div>'
        + paramsHtml
        + '<div class="row"><input id="p-name-'+k.id+'" placeholder="param name" style="width:140px"/> '
        + '<input type="number" id="p-val-'+k.id+'" placeholder="0" style="width:80px" min="0" max="100"/> % '
        + '<button onclick="createParam(\''+k.id+'\')">Add param</button></div>'
        + '</div>');
    }
    const usedBadge = f.used
      ? '<span class="badge ok">Used</span>'
      : '<span class="badge muted">Not used</span>';
    const svcHtml = '<div class="svc-list">'
      + (f.services||[]).map(s=>'<span class="svc" title="'+(s && s.name ? s.name : '')+'">'+(s && s.name ? s.name : '')+' <button class="mini" onclick="unlinkService(\''+f.id+'\',\''+(s && s.id ? s.id : '')+'\')">✖</button></span>').join(' ')
      + '</div>'
      + '<div class="row"><select id="svc-'+f.id+'" multiple size="'+Math.min(5, (availableServices||[]).length||1)+'" style="flex:1">'
      + availableServices.map(s=>'<option value="'+s.id+'">'+s.name+'</option>').join('')
      + '</select> <button onclick="linkServiceMulti(\''+f.id+'\')">Link selected</button></div>';

    tr.innerHTML = '<td><div><b>' + f.name + '</b></div></td>'
      + '<td>' + (f.description||'') + '</td>'
      + '<td><input type="number" id="val-' + f.id + '" value="' + (f.value||0) + '" class="value-input" min="0" max="100"/> %</td>'
      + '<td>' + usedBadge + '</td>'
      + '<td>' + svcHtml + '</td>'
      + '<td>' + keysHtmlResolved.join('') + '<div class="row"><input id="k-name-'+f.id+'" placeholder="key name" style="width:140px"/> '
      + '<input type="number" id="k-val-'+f.id+'" placeholder="0" style="width:80px" min="0" max="100"/> % '
      + '<button onclick="createKey(\''+f.id+'\')">Add key</button></div></td>'
      + '<td>'
      + '  <div class="actions">'
      + '    <button onclick="setValue(\'' + f.id + '\')">Save</button>'
      + '    <button class="danger" onclick="del(\'' + f.id + '\')">Delete</button>'
      + '  </div>'
      + '</td>';
    tbody.appendChild(tr);
  }
}

function onSearchInput(){
  // debounce not needed for simple filtering; re-render
  fetchFeatures();
}

// helper lookups
function findFeatureById(id){
  return (__state.features||[]).find(f=>f.id===id);
}
function findFeatureAndKeyByKeyId(keyId){
  for(const f of (__state.features||[])){
    const k = (f.keys||[]).find(x=>x.id===keyId);
    if(k){ return {feature:f, key:k}; }
  }
  return {feature:null, key:null};
}
function findParamContext(paramId){
  for(const f of (__state.features||[])){
    for(const k of (f.keys||[])){
      const p = (k.params||[]).find(x=>x.id===paramId);
      if(p){ return {feature:f, key:k, param:p}; }
    }
  }
  return {feature:null, key:null, param:null};
}

async function createFeature(){
  const name = document.getElementById('f-name').value.trim();
  const description = document.getElementById('f-desc').value.trim();
  if(!name){return}
  const res = await fetch('/api/features',{method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({name, description})});
  if(res.ok){ document.getElementById('f-name').value=''; document.getElementById('f-desc').value=''; await fetchFeatures(); }
}

async function setValue(id){
  let v = parseInt(document.getElementById('val-'+id).value||'0',10);
  v = clampPercent(v);
  const f = findFeatureById(id);
  if(!f){ return }
  const body = { name: f.name||'', description: f.description||'', value: v };
  const res = await fetch('/api/features/'+id,{method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body)});
  if(res.ok){ await fetchFeatures(); }
}

async function setKeyValue(id){
  let v = parseInt(document.getElementById('k-value-'+id).value||'0',10);
  v = clampPercent(v);
  const ctx = findFeatureAndKeyByKeyId(id);
  if(!ctx.feature || !ctx.key){ return }
  const body = { feature_id: ctx.feature.id, key: ctx.key.name||'', value: v };
  const res = await fetch('/api/keys/'+id,{method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body)});
  if(res.ok){ await fetchFeatures(); }
}

async function del(id){
  if(!confirm('Delete feature?')) return;
  const res = await fetch('/api/features/'+id,{method:'DELETE'});
  if(res.ok){ await fetchFeatures(); }
}

async function createKey(featureId){
  const key = document.getElementById('k-name-'+featureId).value.trim();
  let value = parseInt(document.getElementById('k-val-'+featureId).value||'0',10);
  value = clampPercent(value);
  if(!key){return}
  const res1 = await fetch('/api/features/'+featureId+'/keys',{method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({key, value})});
  if(res1.ok){
    document.getElementById(`k-name-${featureId}`).value='';
    document.getElementById(`k-val-${featureId}`).value='';
    await fetchFeatures();
  }
}

async function deleteKey(id){
  const res = await fetch('/api/keys/'+id,{method:'DELETE'});
  if(res.ok){ await fetchFeatures(); }
}

async function fetchServices(){
  const res = await fetch('/api/services');
  if(!res.ok){return []}
  return await res.json();
}

async function createService(){
  const name = document.getElementById('s-name').value.trim();
  if(!name){return}
  const res = await fetch('/api/services',{method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({name})});
  if(res.ok){ document.getElementById('s-name').value=''; await fetchFeatures(); }
}

async function linkService(featureId){
  const sid = document.getElementById('svc-'+featureId).value;
  if(!sid){return}
  const res = await fetch('/api/features/'+featureId+'/services/'+sid,{method:'POST'});
  if(res.ok){ await fetchFeatures(); }
}

async function linkServiceMulti(featureId){
  const select = document.getElementById('svc-'+featureId);
  if(!select){ return }
  const ids = Array.from(select.selectedOptions||[]).map(o=>o.value);
  for(const sid of ids){
    await fetch('/api/features/'+featureId+'/services/'+sid,{method:'POST'});
  }
  await fetchFeatures();
}

async function unlinkService(featureId, serviceId){
  const res = await fetch('/api/features/'+featureId+'/services/'+serviceId,{method:'DELETE'});
  if(res.ok){ await fetchFeatures(); }
}

function renderServicesSidebar(services){
  const wrap = document.getElementById('services');
  if(!wrap) return;
  wrap.innerHTML = services.map(s=>{
    const activeBadge = s.active
      ? '<span class="badge ok">Active</span>'
      : '<span class="badge muted">Inactive</span>';
    const delBtn = s.active ? '' : ' <button class="danger" onclick="deleteService(\''+s.id+'\')">Delete</button>';
    return '<div class="svc-item"><div><b>'+s.name+'</b> '+activeBadge+'</div><div>'+delBtn+'</div></div>';
  }).join('');
}

async function deleteService(id){
  if(!confirm('Delete service?')) return;
  const res = await fetch('/api/services/'+id,{method:'DELETE'});
  if(res.ok){ await fetchFeatures(); }
}

async function createParam(keyId){
  const name = document.getElementById('p-name-'+keyId).value.trim();
  let value = parseInt(document.getElementById('p-val-'+keyId).value||'0',10);
  value = clampPercent(value);
  if(!name){return}
  const ctx = findFeatureAndKeyByKeyId(keyId);
  if(!ctx.feature){ return }
  const res1 = await fetch('/api/keys/'+keyId+'/params',{method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({feature_id: ctx.feature.id, name, value})});
  if(res1.ok){
    document.getElementById('p-name-'+keyId).value='';
    document.getElementById('p-val-'+keyId).value='';
    await fetchFeatures();
  }
}

async function setParamValue(id){
  let v = parseInt(document.getElementById('p-value-'+id).value||'0',10);
  v = clampPercent(v);
  const ctx = findParamContext(id);
  if(!ctx.feature || !ctx.key || !ctx.param){ return }
  const body = { feature_id: ctx.feature.id, key_id: ctx.key.id, name: ctx.param.name||'', value: v };
  const res = await fetch('/api/params/'+id,{method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body)});
  if(res.ok){ await fetchFeatures(); }
}

async function deleteParam(id){
  const res = await fetch('/api/params/'+id,{method:'DELETE'});
  if(res.ok){ await fetchFeatures(); }
}

function clampPercent(v){
  if(isNaN(v)) return 0;
  if(v < 0) return 0;
  if(v > 100) return 100;
  return v|0;
}

window.addEventListener('load', fetchFeatures);
