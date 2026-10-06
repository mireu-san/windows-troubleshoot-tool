import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
import { build } from 'vite';
import { JSDOM, VirtualConsole } from 'jsdom';

const waitFor = async predicate => {
  for (let i = 0; i < 100; i++) {
    if (predicate()) return;
    await new Promise(resolve => setTimeout(resolve, 10));
  }
  assert.fail('UI did not reach expected state');
};

test('missing-source recovery handles dialog cancel, fast events, completion and Windows recovery', async () => {
  const bundle = await build({root:fileURLToPath(new URL('..',import.meta.url)),configFile:false,logLevel:'silent',build:{write:false,minify:false}});
  const code = bundle.output.find(item => item.type === 'chunk' && item.isEntry).code;
  const errors = [];
  const virtualConsole = new VirtualConsole();
  virtualConsole.on('jsdomError', error => errors.push(error));
  const dom = new JSDOM('<!doctype html><html><body><div id="app"></div></body></html>', {url:'http://localhost',runScripts:'dangerously',pretendToBeVisual:true,virtualConsole});
  const {window} = dom;
  const events = {};
  let selected = false, recoveryCalls = 0;
  window.go = {main:{App:{
    GetLanguageSettings:async()=>({preference:'ko',language:'ko'}),
    GetAppVersion:async()=> 'test', CheckAppUpdate:async()=>({available:false}), IsRunning:async()=>false,
    StartRepairFromMedia:async()=> {
      if (!selected) return false;
      events['repair:event']({stage:'source',title:'Checking source',message:'Checking'});
      return true;
    },
    OpenWindowsRepairSettings:async()=> { recoveryCalls++; },
  }}};
  window.runtime = {EventsOn:(name, handler)=>{events[name]=handler}};
  window.eval(code);
  const $ = selector=>window.document.querySelector(selector);
  try {
    await waitFor(()=>events['repair:event']);
    const fail = ()=>events['repair:event']({stage:'error',title:'Missing source',message:'Error',sourceRequired:true,debugAvailable:true,sourceDetails:'build mismatch <img src=x onerror=alert(1)>'});
    assert.equal($('#repairSourcePanel').hidden,true);
    fail();
    assert.equal($('#repairSourcePanel').hidden,false);
    assert.equal($('#sourceDetails').textContent, 'build mismatch <img src=x onerror=alert(1)>');
    assert.equal($('#sourceDetails img'), null);
    $('#repairFromMedia').click();
    await waitFor(()=>!$('#repairFromMedia').disabled);
    assert.equal($('#repairSourcePanel').hidden,false,'dialog cancellation preserves recovery actions');
    assert.equal($('#startButton').disabled,false);
    $('#openRepairSettings').click();
    await waitFor(()=>recoveryCalls===1);
    await waitFor(()=>$('#repairSourceStatus').textContent.includes('아직 복구 성공'));
    assert.equal($('#repairSourcePanel').hidden, false);
    selected = true;
    $('#repairFromMedia').click();
    await waitFor(()=>!$('#repairFromMedia').disabled);
    assert.equal($('#repairSourcePanel').hidden,true);
    assert.equal($('#sourceDetails').textContent, '');
    assert.equal($('#startButton').disabled,true,'early source event is not overwritten');
    assert.equal($('#cancelRepairButton').hidden,false);
    events['repair:event']({stage:'complete',title:'Complete',message:'Done',progress:100});
    assert.equal($('#startButton').disabled,false);
    assert.equal($('#repairSourcePanel').hidden,true);
    fail();
    assert.equal($('#repairSourcePanel').hidden,false);
    events['repair:event']({stage:'starting',title:'Start',message:'Start'});
    assert.equal($('#repairSourcePanel').hidden,true);
    assert.deepEqual(errors,[]);
  } finally {window.close();}
});
