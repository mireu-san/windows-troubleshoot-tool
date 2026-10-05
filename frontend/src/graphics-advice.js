export function graphicsAdvice({ taskManager, scope, recent }) {
  const advice = [];
  if (taskManager === 'yes') advice.push('작업 관리자도 깜박이면 디스플레이 드라이버 문제를 의심할 단서입니다. 자동 검사 결과와 함께 확인하세요.');
  if (taskManager === 'no') advice.push('작업 관리자는 정상이고 다른 화면만 깜박이면 앱 호환성 문제를 의심할 단서입니다.');
  if (scope === 'app') advice.push('해당 앱을 업데이트하고 하드웨어 가속 설정을 한 번에 하나씩 바꿔 재현 여부를 비교하세요.');
  if (scope === 'external') advice.push('외부 모니터의 케이블·포트·도킹 장치를 바꿔 비교하고, 고급 디스플레이 설정에서 주사율을 확인하세요.');
  if (recent === 'yes') advice.push('업데이트 직후 시작됐다면 장치 관리자에서 이전 드라이버로 되돌리기가 가능한지 확인하세요.');
  if (!advice.length) advice.push('깜박일 때 작업 관리자도 영향을 받는지, 특정 앱이나 모니터에서만 발생하는지 확인해 주세요.');
  advice.push('이 안내는 원인을 좁히기 위한 단서이며 Intel 내장 그래픽이나 GPU 하드웨어 고장을 확정하지 않습니다.');
  return advice;
}
