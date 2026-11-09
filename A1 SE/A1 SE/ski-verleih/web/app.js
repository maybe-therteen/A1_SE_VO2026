const state = {
    stats: { availableSkis: 0, rentedSkis: 0, totalSkis: 0, registeredPersons: 0 },
    skis: [],
    persons: [],
    reservations: [],
};

document.addEventListener("DOMContentLoaded", () => {
    setupNavigation();
    setupFormsAndModal();
    refreshAll();
});

function setupNavigation() {
    document.querySelectorAll("[data-nav]").forEach((el) => {
        el.addEventListener("click", (e) => {
            e.preventDefault();
            const target = el.getAttribute("data-nav");
            switchView(target);
        });
    });
}

function switchView(viewName) {
    document.querySelectorAll(".view").forEach((v) => v.classList.remove("active-view"));
    const targetView = document.getElementById(`view-${viewName}`);
    if (targetView) {
        targetView.classList.add("active-view");
    }

    document.querySelectorAll(".nav-link").forEach((link) => {
        link.classList.toggle("active", link.getAttribute("data-nav") === viewName);
    });
}

function setupFormsAndModal() {
    const modal = document.getElementById("modal-rental");
    const openBtns = [
        document.getElementById("btn-open-rental-modal"),
        document.getElementById("btn-open-rental-modal-2"),
    ];
    openBtns.forEach((btn) => {
        if (btn) {
            btn.addEventListener("click", () => openRentalModal());
        }
    });

    document.getElementById("btn-close-modal").addEventListener("click", () => modal.classList.add("hidden"));
    document.getElementById("btn-cancel-modal").addEventListener("click", () => modal.classList.add("hidden"));

    // Ski anlegen
    document.getElementById("form-add-ski").addEventListener("submit", async (e) => {
        e.preventDefault();
        const number = document.getElementById("ski-number").value.trim();
        const type = document.getElementById("ski-type").value;
        const lengthCm = parseInt(document.getElementById("ski-length").value, 10);

        try {
            await apiRequest("/api/skis", {
                method: "POST",
                body: JSON.stringify({ number, type, lengthCm }),
            });
            document.getElementById("ski-number").value = "";
            showToast(`Ski ${number} (${type}) erfolgreich angelegt.`);
            await refreshAll();
        } catch (err) {
            showToast(err.message, true);
        }
    });

    // Person anlegen
    document.getElementById("form-add-person").addEventListener("submit", async (e) => {
        e.preventDefault();
        const name = document.getElementById("person-name").value.trim();
        const email = document.getElementById("person-email").value.trim();
        const phone = document.getElementById("person-phone").value.trim();

        try {
            await apiRequest("/api/persons", {
                method: "POST",
                body: JSON.stringify({ name, email, phone }),
            });
            e.target.reset();
            showToast(`Person "${name}" wurde registriert.`);
            await refreshAll();
        } catch (err) {
            showToast(err.message, true);
        }
    });

    // Neue Ausleihe starten
    document.getElementById("form-new-rental").addEventListener("submit", async (e) => {
        e.preventDefault();
        const personId = parseInt(document.getElementById("rental-person").value, 10);
        const skiId = parseInt(document.getElementById("rental-ski").value, 10);
        const notes = document.getElementById("rental-notes").value.trim();

        try {
            const created = await apiRequest("/api/reservations", {
                method: "POST",
                body: JSON.stringify({ skiId, personId, notes }),
            });
            modal.classList.add("hidden");
            document.getElementById("rental-notes").value = "";
            showToast(`Ausleihe #${created.id}: ${created.skiNumber} (${created.skiType}) an ${created.personName} vergeben.`);
            await refreshAll();
        } catch (err) {
            showToast(err.message, true);
        }
    });

    // Filter-Events
    document.getElementById("filter-ski-search").addEventListener("input", renderSkis);
    document.getElementById("filter-ski-status").addEventListener("change", renderSkis);
    document.getElementById("filter-res-search").addEventListener("input", renderReservations);
    document.getElementById("filter-res-status").addEventListener("change", renderReservations);
}

async function apiRequest(url, options = {}) {
    const res = await fetch(url, {
        headers: { "Content-Type": "application/json" },
        ...options,
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
        throw new Error(data.error || `HTTP-Fehler ${res.status}`);
    }
    return data;
}

async function refreshAll() {
    try {
        const [stats, skis, persons, reservations] = await Promise.all([
            apiRequest("/api/dashboard"),
            apiRequest("/api/skis"),
            apiRequest("/api/persons"),
            apiRequest("/api/reservations"),
        ]);
        state.stats = stats;
        state.skis = skis;
        state.persons = persons;
        state.reservations = reservations;

        renderDashboard();
        renderSkis();
        renderPersons();
        renderReservations();
    } catch (err) {
        showToast("Fehler beim Laden der Serverdaten: " + err.message, true);
    }
}

function renderDashboard() {
    document.getElementById("stat-available").textContent = state.stats.availableSkis;
    document.getElementById("stat-rented").textContent = state.stats.rentedSkis;
    document.getElementById("stat-total").textContent = state.stats.totalSkis;
    document.getElementById("stat-persons").textContent = state.stats.registeredPersons;

    const tbody = document.getElementById("dashboard-active-rentals-body");
    const active = state.reservations.filter((r) => r.active);

    if (active.length === 0) {
        tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;color:#64748b;">Derzeit sind keine Ski ausgeliehen.</td></tr>`;
        return;
    }

    tbody.innerHTML = active
        .map(
            (r) => `
        <tr>
            <td><strong>#${r.id}</strong></td>
            <td><strong>${escapeHtml(r.skiNumber)}</strong> <span class="badge badge-type">${escapeHtml(r.skiType)} (${r.skiLength} cm)</span></td>
            <td>${escapeHtml(r.personName)}</td>
            <td>${formatDateTime(r.startTime)}</td>
            <td>${escapeHtml(r.notes || "–")}</td>
            <td>
                <button class="btn-return-sm" onclick="returnSki(${r.id})">Ski zurückgeben</button>
            </td>
        </tr>
    `
        )
        .join("");
}

function renderSkis() {
    const search = document.getElementById("filter-ski-search").value.trim().toLowerCase();
    const statusFilter = document.getElementById("filter-ski-status").value;

    // Aktive Ausleihe pro Ski ermitteln für transparente Nachvollziehbarkeit
    const activeBySkiId = {};
    state.reservations.forEach((r) => {
        if (r.active) {
            activeBySkiId[r.skiId] = r;
        }
    });

    const filtered = state.skis.filter((s) => {
        if (statusFilter === "available" && !s.available) return false;
        if (statusFilter === "rented" && s.available) return false;
        if (search) {
            const hay = `${s.number} ${s.type} ${s.lengthCm}`.toLowerCase();
            if (!hay.includes(search)) return false;
        }
        return true;
    });

    const tbody = document.getElementById("skis-table-body");
    if (filtered.length === 0) {
        tbody.innerHTML = `<tr><td colspan="7" style="text-align:center;color:#64748b;">Keine passenden Ski gefunden.</td></tr>`;
        return;
    }

    tbody.innerHTML = filtered
        .map((s) => {
            const activeRes = activeBySkiId[s.id];
            const statusBadge = s.available
                ? `<span class="badge badge-available">Verfügbar</span>`
                : `<span class="badge badge-rented">Ausgeliehen</span>`;
            const borrowerInfo = activeRes
                ? `${escapeHtml(activeRes.personName)} <small style="color:#64748b;">(seit ${formatDateTime(activeRes.startTime)})</small>`
                : "–";
            const actionBtn = s.available
                ? `<button class="btn-danger-sm" onclick="deleteSki(${s.id})">Löschen</button>`
                : `<button class="btn-return-sm" onclick="returnSki(${activeRes.id})">Rückgabe buchen</button>`;

            return `
            <tr>
                <td>${s.id}</td>
                <td><strong>${escapeHtml(s.number)}</strong></td>
                <td><span class="badge badge-type">${escapeHtml(s.type)}</span></td>
                <td>${s.lengthCm} cm</td>
                <td>${statusBadge}</td>
                <td>${borrowerInfo}</td>
                <td>${actionBtn}</td>
            </tr>
        `;
        })
        .join("");
}

function renderPersons() {
    const activeCountByPerson = {};
    state.reservations.forEach((r) => {
        if (r.active) {
            activeCountByPerson[r.personId] = (activeCountByPerson[r.personId] || 0) + 1;
        }
    });

    const tbody = document.getElementById("persons-table-body");
    if (state.persons.length === 0) {
        tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;color:#64748b;">Keine Personen registriert.</td></tr>`;
        return;
    }

    tbody.innerHTML = state.persons
        .map((p) => {
            const count = activeCountByPerson[p.id] || 0;
            return `
            <tr>
                <td>${p.id}</td>
                <td><strong>${escapeHtml(p.name)}</strong></td>
                <td>${escapeHtml(p.email)}</td>
                <td>${escapeHtml(p.phone || "–")}</td>
                <td>${count > 0 ? `<span class="badge badge-rented">${count} aktiv</span>` : `<span class="badge badge-returned">0 aktiv</span>`}</td>
                <td>
                    <button class="btn-danger-sm" onclick="deletePerson(${p.id})">Löschen</button>
                </td>
            </tr>
        `;
        })
        .join("");
}

function renderReservations() {
    const search = document.getElementById("filter-res-search").value.trim().toLowerCase();
    const statusFilter = document.getElementById("filter-res-status").value;

    const filtered = state.reservations.filter((r) => {
        if (statusFilter === "active" && !r.active) return false;
        if (statusFilter === "returned" && r.active) return false;
        if (search) {
            const hay = `${r.skiNumber} ${r.skiType} ${r.personName} ${r.notes || ""}`.toLowerCase();
            if (!hay.includes(search)) return false;
        }
        return true;
    });

    const tbody = document.getElementById("reservations-table-body");
    if (filtered.length === 0) {
        tbody.innerHTML = `<tr><td colspan="9" style="text-align:center;color:#64748b;">Keine Reservierungen für diesen Filter vorhanden.</td></tr>`;
        return;
    }

    tbody.innerHTML = filtered
        .map((r) => {
            const badge = r.active
                ? `<span class="badge badge-rented">Aktiv ausgeliehen</span>`
                : `<span class="badge badge-returned">Zurückgegeben</span>`;
            const action = r.active
                ? `<button class="btn-return-sm" onclick="returnSki(${r.id})">Rückgabe</button>`
                : `–`;
            return `
            <tr>
                <td><strong>#${r.id}</strong></td>
                <td><strong>${escapeHtml(r.skiNumber)}</strong></td>
                <td><span class="badge badge-type">${escapeHtml(r.skiType)}</span> (${r.skiLength} cm)</td>
                <td>${escapeHtml(r.personName)}</td>
                <td>${formatDateTime(r.startTime)}</td>
                <td>${r.endTime ? formatDateTime(r.endTime) : "<em>noch ausgeliehen</em>"}</td>
                <td>${badge}</td>
                <td>${escapeHtml(r.notes || "–")}</td>
                <td>${action}</td>
            </tr>
        `;
        })
        .join("");
}

function openRentalModal() {
    const availableSkis = state.skis.filter((s) => s.available);
    if (availableSkis.length === 0) {
        showToast("Aktuell sind keine verfügbaren Ski im Bestand!", true);
        return;
    }
    if (state.persons.length === 0) {
        showToast("Bitte zuerst mindestens eine Person registrieren!", true);
        return;
    }

    const personSelect = document.getElementById("rental-person");
    personSelect.innerHTML = state.persons
        .map((p) => `<option value="${p.id}">${escapeHtml(p.name)} (${escapeHtml(p.email)})</option>`)
        .join("");

    const skiSelect = document.getElementById("rental-ski");
    skiSelect.innerHTML = availableSkis
        .map((s) => `<option value="${s.id}">${escapeHtml(s.number)} – ${escapeHtml(s.type)} (${s.lengthCm} cm)</option>`)
        .join("");

    document.getElementById("modal-rental").classList.remove("hidden");
}

async function returnSki(reservationId) {
    try {
        const res = await apiRequest(`/api/reservations/${reservationId}/return`, { method: "POST" });
        showToast(`Ski ${res.skiNumber} (${res.skiType}) wurde von ${res.personName} zurückgegeben.`);
        await refreshAll();
    } catch (err) {
        showToast(err.message, true);
    }
}

async function deleteSki(skiId) {
    try {
        await apiRequest(`/api/skis/${skiId}`, { method: "DELETE" });
        showToast(`Ski #${skiId} wurde gelöscht.`);
        await refreshAll();
    } catch (err) {
        showToast(err.message, true);
    }
}

async function deletePerson(personId) {
    try {
        await apiRequest(`/api/persons/${personId}`, { method: "DELETE" });
        showToast(`Person #${personId} wurde gelöscht.`);
        await refreshAll();
    } catch (err) {
        showToast(err.message, true);
    }
}

function formatDateTime(iso) {
    if (!iso) return "–";
    const d = new Date(iso);
    return d.toLocaleString("de-AT", {
        day: "2-digit",
        month: "2-digit",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
    });
}

function escapeHtml(str) {
    return String(str)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
}

let toastTimer = null;
function showToast(msg, isError = false) {
    const toast = document.getElementById("toast");
    toast.textContent = msg;
    toast.classList.toggle("error", isError);
    toast.classList.remove("hidden");
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(() => {
        toast.classList.add("hidden");
    }, 3600);
}
