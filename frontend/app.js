const API = "http://localhost:8080";

let token = "";
let socket = null;

async function register() {
    const username = document.getElementById("regUsername").value;
    const password = document.getElementById("regPassword").value;

    const res = await fetch(`${API}/auth/register`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({ username, password })
    });

    const data = await res.json();

    alert(JSON.stringify(data));
}

async function login() {
    const username = document.getElementById("loginUsername").value;
    const password = document.getElementById("loginPassword").value;

    const res = await fetch(`${API}/auth/login`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({ username, password })
    });

    const data = await res.json();

    token = data.token;

    alert("Login success!");

    connectWebSocket();
}

async function searchManga() {
    const query = document.getElementById("searchInput").value;

    const res = await fetch(`${API}/manga?q=${query}`);

    const data = await res.json();

    document.getElementById("mangaResults").innerHTML =
        JSON.stringify(data, null, 2);
}

function connectWebSocket() {
    socket = new WebSocket("ws://localhost:8080/ws");

    socket.onmessage = (event) => {
        const chatBox = document.getElementById("chatBox");

        chatBox.innerHTML += `<p>${event.data}</p>`;
    };
}

function sendMessage() {
    const input = document.getElementById("chatInput");

    socket.send(input.value);

    input.value = "";
}