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

    if (!res.ok) {
        alert(data.error || "Login failed");
        return;
    }

    token = data.token;

    alert("Login success!");

    connectWebSocket();
}

async function searchManga() {
    const query = document.getElementById("searchInput").value;

    const res = await fetch(`${API}/manga?query=${query}`);

    const data = await res.json();

    document.getElementById("mangaResults").innerHTML =
        JSON.stringify(data.results, null, 2);
}
