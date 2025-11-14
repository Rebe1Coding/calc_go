const output = document.getElementById('output');
const input = document.getElementById('input');

function sendCommand() {
  const cmd = input.value.trim();
  if (!cmd) return;

  fetch('/api/execute', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ input: cmd })
  })
  .then(r => r.json())
  .then(data => {
    if (data.error) {
      output.innerHTML += '\n❌ Ошибка: ' + data.error;
    } else {
      output.innerHTML += '\n📊 Результат: ' + JSON.stringify(data.result);
    }
    input.value = '';
  });
}

function showVars() {
  fetch('/api/vars').then(r => r.json()).then(vars => {
    output.innerHTML += '\n📊 Переменные: ' + JSON.stringify(vars);
  });
}

function showHistory() {
  fetch('/api/history').then(r => r.json()).then(history => {
    output.innerHTML += '\n📜 История:\n' + history.join('\n');
  });
}

function clearOutput() {
  output.innerHTML = '';
}

input.addEventListener('keypress', e => {
  if (e.key === 'Enter') sendCommand();
});