document.addEventListener("DOMContentLoaded", () => {
    document.querySelectorAll(".progress").forEach(td => {
        const current = parseInt(td.dataset.current);
        const total = parseInt(td.dataset.total);

        if (!total || total === 0) {
            td.innerHTML = "0%";
            return;
        }

        const percent = Math.floor((current / total) * 100);
        td.innerHTML = `
            <div style="background:#333;border-radius:10px;overflow:hidden;">
                <div style="width:${percent}%;background:#4caf50;padding:5px;text-align:center;color:white;">
                    ${percent}%
                </div>
            </div>`;
    });
});

async function updateEp(id, action) {
    await fetch(`/update?id=${id}&action=${action}`, { method: "POST" });
    location.reload();
}

async function deleteSeries(id) {
    if (!confirm("Delete this series?")) return;
    await fetch("/delete?id=" + id, { method: "DELETE" });
    location.reload();
}

async function rate(id, value) {
    await fetch(`/rate?id=${id}&value=${value}`, { method: "POST" });
    location.reload();
}