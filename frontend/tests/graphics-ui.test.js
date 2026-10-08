import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
import { build } from 'vite';
import { JSDOM, VirtualConsole } from 'jsdom';

const waitFor = async predicate => {
  for (let i=0;i<100;i++) {
    if (predicate()) return;
    await new Promise(resolve => setTimeout(resolve,10));
  }
  assert.fail('UI did not reach expected state');
};

test('graphics UI boots after language init, routes updates, marks symptoms and saves reports', async () => {
  const bundle = await build({root:fileURLToPath(new URL('..',import.meta.url)),configFile:false,logLevel:'silent',build:{write:false,minify:false}});
  const code = bundle.output.find(item => item.type==='chunk' && item.isEntry).code;
  const errors = [];
  const virtualConsole = new VirtualConsole();
  virtualConsole.on('jsdomError',error => errors.push(error));
  const dom = new JSDOM('<!doctype html><html><body><div id="app"></div></body></html>',{url:'http://localhost',runScripts:'dangerously',pretendToBeVisual:true,virtualConsole});
  const {window} = dom;
  const calls = [], events = {};
  const report = {manufacturer:'Example',model:'Hybrid',from:'2026-10-05T12:00:00Z',until:'2026-10-05T12:02:00Z',gpus:[{name:'Intel GPU',vendor:'intel',version:'1',problemCode:0},{name:'NVIDIA GPU',vendor:'nvidia',version:'2'}],monitors:[],findings:[{title:'관련 오류 근거를 찾지 못했습니다',evidence:'<img src=x onerror=alert(1)>',action:'확인'}],issues:['partial collection'],events:[]};
  window.go={main:{App:{
    GetLanguageSettings:async()=>({preference:'ko',language:'ko'}),
    GetAppVersion:async()=> 'test',CheckAppUpdate:async()=>({available:false}),IsRunning:async()=>false,
    InvestigateHardware:async request=>{calls.push({hardware:request});return {folder:'C:\\Desktop\\하드웨어진단_test',summary:'메인보드 확정 근거 부족'}},
    SaveHardwareReport:async()=>({folder:'C:\\Desktop\\하드웨어진단_retry',summary:'다시 저장'}),
    OpenHardwareFolder:async()=>{calls.push('hardware-open')},
    ScanGraphics:async()=>{calls.push('scan');return report},
    MarkGraphicsFlicker:async()=>{calls.push('mark');return {...report,marker:'2026-10-05T12:01:00Z'}},
    StartGraphicsUpdate:async vendor=>{calls.push(vendor);return {message:'제조사 업데이트 도구를 실행했습니다. 도구에서 설치를 마친 뒤 다시 검사해 주세요.'}},
    ExportGraphicsReport:async answers=>{calls.push(JSON.parse(answers));return 'C:\\reports\\graphics.json'},
    ScanFlickerDiagnostics:async()=>{calls.push('comprehensive');return {...report, comprehensive:{cpu:[{Name:'CPU'}],issues:['Security access denied']}}},
    ExportFlickerReport:async answers=>{calls.push(JSON.parse(answers));return ''},
    StartDisplayWatch:async()=>({running:true,changes:[],markers:[]}),
    GetDisplayWatch:async()=>({running:true,changes:[],markers:[]}),
    StopDisplayWatch:async()=>({running:false,changes:[{before:{width:1920},after:{width:1280}}],markers:[]}),
    ExportRepairDiagnostics:async()=> 'C:\\reports\\repair.txt',
  }}};
  window.runtime={EventsOn:(name,handler)=>{events[name]=handler}};
  window.eval(code);
  const $=selector=>window.document.querySelector(selector);
  await waitFor(()=>$('#scanGraphicsButton'));
  assert.equal($('#hardwareOpen').disabled,true);
  $('#hardwareWiring').value='HDMI → monitor';
  $('#hardwareSymptoms').value='신호 없음';
  $('#hardwareInvestigate').click();
  await waitFor(()=>!$('#hardwareInvestigate').disabled && calls.some(c=>c.hardware));
  assert.equal(calls.find(c=>c.hardware).hardware.wiring,'HDMI → monitor');
  assert.match($('#hardwareStatus').textContent,/하드웨어진단_test/);
  $('#hardwareOpen').click();
  await waitFor(()=>calls.includes('hardware-open') && !$('#hardwareOpen').disabled);
  $('#hardwareSave').click();
  await waitFor(()=>$('#hardwareStatus').textContent.includes('하드웨어진단_retry') && !$('#hardwareSave').disabled);
  $('#scanGraphicsButton').click();
  await waitFor(()=>$('#graphicsResults') && !$('#graphicsResults').hidden && !$('#scanGraphicsButton').disabled);
  assert.equal($('#graphicsDevices').querySelectorAll('article').length,2);
  assert.equal($('#graphicsDevices').querySelectorAll('button').length,2);
  assert.equal($('#graphicsFindings').querySelector('img'),null,'event strings must not become HTML');
  $('#graphicsDevices').querySelectorAll('button')[1].click();
  await waitFor(()=>calls.includes('nvidia')&&!$('#scanGraphicsButton').disabled);
  assert.match($('#graphicsStatus').textContent,/도구/);
  $('#markFlickerButton').click();
  await waitFor(()=>calls.includes('mark')&&!$('#scanGraphicsButton').disabled);
  $('#graphicsScope').value='external';
  $('#graphicsScope').dispatchEvent(new window.Event('change'));
  assert.match($('#graphicsAdvice').textContent,/케이블/);
  $('#saveGraphicsButton').click();
  await waitFor(()=>$('#graphicsStatus').textContent.includes('graphics.json'));
  assert.equal(calls.at(-1).scope,'external');
  assert.equal($('#stopDisplayWatch').disabled,true);
  assert.equal($('#saveFlickerButton').disabled,true);
  $('#flickerRecovery').value='DISM/SFC 이후';
  $('#flickerTiming').value='separate';
  $('#flickerDiagnosticsButton').click();
  await waitFor(()=>!$('#flickerDiagnosticsButton').disabled && calls.includes('comprehensive'));
  assert.match($('#graphicsStatus').textContent,/취소/);
  assert.equal(calls.at(-1).recovery,'DISM/SFC 이후');
  assert.equal(calls.at(-1).timing,'separate');
  assert.match($('#flickerRaw').textContent,/Security access denied/);
  assert.equal($('#saveFlickerButton').disabled,false);
  $('#startDisplayWatch').click();
  await waitFor(()=>!$('#stopDisplayWatch').disabled);
  assert.equal($('#startDisplayWatch').disabled,true);
  $('#stopDisplayWatch').click();
  await waitFor(()=>!$('#startDisplayWatch').disabled);
  assert.equal($('#stopDisplayWatch').disabled,true);
  assert.match($('#displayWatchRaw').textContent,/1280/);
  window.go.main.App.ExportFlickerReport=async()=>{throw new Error('disk full')};
  $('#saveFlickerButton').click();
  await waitFor(()=>$('#graphicsStatus').textContent.includes('disk full'));
  assert.equal($('#saveFlickerButton').disabled,false);
  events['repair:event']({stage:'error',debugAvailable:true,title:'Failed',message:'Error'});
  assert.equal($('#debugPanel').hidden,false);
  $('#exportDiagnostics').click();
  await waitFor(()=>$('#debugStatus').textContent.includes('repair.txt'));
  assert.deepEqual(errors,[]);
  window.close();
});
