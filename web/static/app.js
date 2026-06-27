document.querySelectorAll('#channels button').forEach(button => {
  button.addEventListener('click', async () => {
    const channel = button.dataset.channel;
    const response = await fetch(`/next?channel=${channel}`);
    const data = await response.json();

    document.getElementById('output').innerText = `[${data.channel}] ${data.text}`;
  });
});
