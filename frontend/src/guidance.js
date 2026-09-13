// Local decision tree: answers guide users to tools; they never run commands.
const question = (title, description, choices) => ({ type: 'question', title, description, choices });
const choice = (value, label, next) => ({ value, label, next });
const result = (title, reason, action, after, limit, target = null, tool = null) => ({
  type: 'result', title, reason, action, after, limit, target, tool,
});

export const guideNodes = {
  symptom: question('어떤 점이 가장 불편한가요?', '지금 해결하고 싶은 문제 하나를 골라 주세요. 잘 모르겠어도 괜찮습니다.', [
    choice('update', 'Windows 업데이트가 설치되지 않거나 멈춰요', 'updateConnection'),
    choice('system', '오류가 반복되거나 Windows 기능이 잘 안 돼요', 'systemScope'),
    choice('security', '수상한 광고나 악성 프로그램이 의심돼요', 'securityTried'),
    choice('boot', '컴퓨터가 켜지지 않거나 다시 시작을 반복해요', 'bootAccess'),
    choice('unsure', '느려졌거나, 무엇이 문제인지 모르겠어요', 'unsureDetail'),
  ]),
  updateConnection: question('문제가 생긴 PC에서 인터넷이 되나요?', '인터넷 창에서 평소 쓰는 웹사이트가 열리는지 확인해 주세요. 업데이트 파일을 받으려면 연결이 필요합니다.', [
    choice('online', '네, 웹사이트가 정상적으로 열려요', 'updateTried'),
    choice('offline', '아니요, 인터넷도 안 돼요', 'network'),
    choice('unknown', '아직 확인하지 못했어요', 'network'),
  ]),
  updateTried: question('업데이트 문제를 해결하려고 어디까지 해봤나요?', '직접 해본 단계까지만 골라 주세요. 잘 모르겠으면 첫 번째를 선택해도 됩니다.', [
    choice('new', '아직 안 해봤거나 잘 모르겠어요', 'update'),
    choice('basic', '문제 해결사를 실행하고 PC도 다시 시작했어요', 'cache'),
    choice('cache', '업데이트 캐시 초기화 후 재시작도 했어요', 'updateRepair'),
    choice('all', '시스템 검사와 복구까지 했는데 그대로예요', 'repeated'),
  ]),
  systemScope: question('어디에서 문제가 생기나요?', '문제가 나타나는 범위에 따라 먼저 확인할 곳이 달라집니다.', [
    choice('windows', '시작 메뉴·설정 등 여러 Windows 기능에서 생겨요', 'repairTried'),
    choice('oneapp', '특정 프로그램 하나에서만 생겨요', 'singleApp'),
    choice('hardware', '전원이 갑자기 꺼지거나, 심한 발열·이상한 소음이 있어요', 'hardware'),
    choice('unknown', '어디서 생기는지 구분하기 어려워요', 'unsure'),
  ]),
  repairTried: question('시스템 검사와 복구를 이미 해봤나요?', '이 앱의 검사·복구가 끝난 뒤 PC를 다시 시작해 보았는지 알려 주세요.', [
    choice('new', '아직 안 했거나 잘 모르겠어요', 'repair'),
    choice('done', '끝까지 실행하고 다시 시작했는데 그대로예요', 'repeated'),
  ]),
  securityTried: question('Windows 보안에서 검사를 해봤나요?', '검사 여부에 따라 새 검사를 시작하거나 기존 결과를 확인할 수 있습니다.', [
    choice('new', '아직 안 해봤어요', 'security'),
    choice('found', '위협이 발견됐거나 조치하라는 안내가 나왔어요', 'securityResults'),
    choice('still', '검사를 했는데 수상한 증상이 계속돼요', 'securityResults'),
  ]),
  bootAccess: question('문제가 생긴 PC의 바탕화면까지 들어갈 수 있나요?', '다른 PC에서 이 안내를 보고 있다면, 고장 난 PC의 상태를 골라 주세요.', [
    choice('desktop', '가끔이라도 바탕화면에 들어가 앱을 실행할 수 있어요', 'repairTried'),
    choice('no', '바탕화면에 전혀 들어가지 못해요', 'recovery'),
  ]),
  unsureDetail: question('가장 가까운 상황을 골라 주세요.', '느려진 이유는 여러 가지입니다. 선택만으로 원인을 확정하지는 않습니다.', [
    choice('errors', '느려지면서 오류도 뜨고 여러 기능이 잘 안 돼요', 'repairTried'),
    choice('oneapp', '특정 프로그램 하나만 느리거나 멈춰요', 'singleApp'),
    choice('slow', '전체적으로 느리지만 특별한 오류는 없어요', 'unsure'),
    choice('unknown', '설명하기 어려워서 누군가 함께 봐줬으면 해요', 'unsure'),
  ]),
  update: result('업데이트 문제 해결부터 시작해 보세요',
    '인터넷은 연결되어 있고, 기본 해결 절차를 아직 시도하지 않았다고 답하셨습니다.',
    'Microsoft의 문제 해결 창을 열어 업데이트 상태를 확인하고 안내에 따라 조치합니다.',
    '안내를 마친 뒤 Windows 업데이트를 다시 시도해 보세요. 실패하면 이 안내에서 해본 단계를 다시 선택해 주세요.',
    '문제 해결 창의 안내를 직접 따라야 합니다. 이 버튼만으로 모든 업데이트 오류가 해결되는 것은 아닙니다.',
    'updateTroubleshooterButton', '업데이트 문제 해결'),
  cache: result('업데이트 캐시 초기화를 다음 단계로 고려해 보세요',
    '인터넷이 되고, 문제 해결사와 재시작을 이미 시도했지만 업데이트가 되지 않는다고 답하셨습니다.',
    '이미 받아 놓은 업데이트 임시 파일을 백업 이름으로 옮겨 Windows가 파일을 새로 받도록 합니다.',
    '초기화가 끝나면 PC를 다시 시작하고 Windows 업데이트를 다시 시도해 보세요.',
    '작업 중 업데이트 관련 서비스를 잠시 멈춥니다. 실행 전 안내를 읽고, 작업이 끝날 때까지 앱을 닫지 마세요.',
    'resetUpdateButton', '업데이트 캐시 초기화'),
  updateRepair: result('Windows 시스템 파일도 검사해 보세요',
    '업데이트 캐시를 초기화하고 다시 시작해도 문제가 계속된다고 답하셨습니다. Windows 파일 손상이 원인인지 확인할 수 있습니다.',
    'Windows 구성 요소를 복구한 다음 시스템 파일을 검사합니다. 손상된 파일이 발견되면 복구를 시도합니다.',
    '검사가 끝나면 필요에 따라 PC를 다시 시작하고 업데이트 설치를 다시 시도해 보세요.',
    '업데이트 실패 원인을 확정한 것은 아닙니다. 같은 문제가 계속되면 오류 코드와 함께 지원을 요청해 주세요.',
    'startButton', '검사 및 복구 시작'),
  repair: result('시스템 검사 및 복구를 시도해 보세요',
    'Windows 사용 중 문제가 있고, 시스템 검사·복구는 아직 완료하지 않았다고 답하셨습니다.',
    'Windows 구성 요소를 먼저 복구하고, 이어서 시스템 파일을 검사합니다. 손상된 파일이 원인인 경우 도움이 될 수 있습니다.',
    '검사가 끝나면 안내에 따라 PC를 다시 시작하고, 처음 문제가 생겼던 기능을 다시 사용해 보세요.',
    '작업에는 10분 이상 걸릴 수 있습니다. 저장장치 고장이나 모든 프로그램 오류를 고치는 도구는 아닙니다.',
    'startButton', '검사 및 복구 시작'),
  security: result('Defender 빠른 검사로 먼저 확인해 보세요',
    '악성 프로그램이 의심되지만 아직 보안 검사를 하지 않았다고 답하셨습니다.',
    'Windows에 포함된 Microsoft Defender로 위협이 자주 발견되는 영역을 검사합니다.',
    '검사 후 「Windows 보안 · 검사 결과 보기」에서 발견된 위협과 필요한 조치를 확인해 주세요.',
    '빠른 검사는 PC의 모든 파일을 검사하지 않습니다. 아무것도 발견되지 않아도 의심 증상이 계속되면 추가 확인이 필요합니다.',
    'defenderQuickScanButton', 'Defender 빠른 검사'),
  securityResults: result('Windows 보안에서 결과와 조치를 확인해 주세요',
    '이미 검사를 했거나 Windows 보안에서 조치가 필요하다는 안내를 받았다고 답하셨습니다.',
    'Windows 보안을 열어 탐지 결과와 필요한 조치를 확인합니다. 필요하면 그 화면에서 추가 검사를 선택할 수 있습니다.',
    'Windows 보안의 안내에 따라 조치한 뒤 같은 증상이 계속되는지 확인해 주세요.',
    '시스템 파일 복구만으로 악성 프로그램이 제거된다고 판단할 수는 없습니다.',
    'defenderSecurityButton', 'Windows 보안 · 검사 결과 보기'),
  singleApp: result('해당 프로그램을 먼저 확인하는 편이 좋겠습니다',
    'Windows 전체가 아니라 특정 프로그램 하나에서만 문제가 생긴다고 답하셨습니다.',
    '프로그램을 다시 열고, 업데이트나 제작사의 지원 안내를 확인해 보세요. 어렵다면 믿을 수 있는 지원자와 화면을 함께 보세요.',
    '같은 프로그램의 같은 작업을 다시 해보세요. 지원을 받을 때는 프로그램 이름과 오류 문구를 알려 주세요.',
    '이 도구는 개별 프로그램의 설정이나 모든 오류를 진단하지 않습니다. 빠른 지원에는 도움을 주는 사람이 필요합니다.',
    'quickAssistButton', '빠른 지원 열기'),
  hardware: result('복구를 반복하기보다 점검을 받아 주세요',
    '갑작스러운 전원 꺼짐, 심한 발열 또는 이상한 소음이 있다고 답하셨습니다.',
    '가능하면 중요한 자료를 먼저 백업하고 PC 제조사나 신뢰하는 기술 지원자에게 점검을 요청하세요.',
    '언제 전원이 꺼지는지, 소음이나 발열이 어디에서 생기는지 전달하면 점검에 도움이 됩니다.',
    '하드웨어 고장 여부는 이 설문으로 확인할 수 없습니다. 시스템 파일 복구나 원격 지원만으로 해결되지 않을 수 있습니다.'),
  unsure: result('빠른 지원으로 함께 확인해 보세요',
    '증상만으로 어떤 복구가 도움이 될지 고르기 어려운 상황입니다.',
    'Microsoft 빠른 지원을 열어 신뢰하는 지원자와 화면을 함께 보면서 문제를 확인할 수 있습니다.',
    '언제부터 느려졌는지, 어떤 작업에서 불편한지 지원자에게 알려 주세요.',
    '이 앱이 지원자를 자동으로 연결해 주지는 않습니다. 알고 있는 사람에게만 화면 공유와 제어를 허용하세요.',
    'quickAssistButton', '빠른 지원 열기'),
  repeated: result('같은 복구를 반복하기보다 원인을 함께 확인해 주세요',
    '시스템 검사와 복구를 마치고 다시 시작해도 문제가 계속된다고 답하셨습니다.',
    '빠른 지원으로 신뢰하는 지원자와 오류 화면, 이미 시도한 작업을 함께 확인할 수 있습니다.',
    '오류 코드와 「자세한 진행 내용 보기」의 결과를 준비해 주세요. 지원자에게 어떤 복구를 했는지 전달해 주세요.',
    '복구 명령의 완료가 원래 문제의 해결을 보장하지는 않습니다. 추가 원인 확인이 필요합니다.',
    'quickAssistButton', '빠른 지원 열기'),
  network: result('먼저 인터넷 연결을 확인해 주세요',
    '인터넷 연결이 되지 않거나 아직 연결 여부를 확인하지 못했다고 답하셨습니다.',
    '평소 쓰는 웹사이트가 열리는지 확인하고, Wi-Fi나 인터넷 케이블 연결을 살펴보세요.',
    '인터넷이 연결된 뒤 이 안내를 다시 열어 업데이트 문제를 선택해 주세요.',
    '업데이트 파일을 받으려면 인터넷이 필요합니다. 연결이 안 되는 상태에서 캐시 초기화부터 권하지 않습니다.'),
  recovery: { ...result('먼저 Windows 부팅 복구 안내를 확인해 주세요',
    '문제가 생긴 PC의 바탕화면에 전혀 들어가지 못한다고 답하셨습니다.',
    'Microsoft의 시동 복구 안내를 참고해 Windows 복구 화면에서 진행해야 합니다. 아래 버튼은 안내 페이지를 엽니다.',
    '복구 화면에 접근하기 어렵거나 중요한 자료가 있다면 PC 제조사 또는 기술 지원자에게 도움을 요청하세요.',
    '현재 앱은 실행되지 않는 다른 PC를 복구하지 못합니다. 장치에 따라 BitLocker 복구 키가 필요할 수 있습니다.'), help: 'recovery' },
};

export function resolveGuide(answers = []) {
  let id = 'symptom';
  const trail = [];
  for (const value of answers) {
    const node = guideNodes[id];
    if (node.type !== 'question') break;
    const selected = node.choices.find(option => option.value === value);
    if (!selected) break;
    trail.push({ question: node.title, answer: selected.label });
    id = selected.next;
  }
  return { ...guideNodes[id], id, trail };
}
