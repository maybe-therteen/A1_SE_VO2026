const state = {
    stats: {},
    bikes: [],
    persons: [],
    reservations: [],
    bikeTypes: {}, // Typ -> Standard-Stundenpreis
};

const eur = new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" });

document.addEventListener("DOMContentLoaded", async () => {
    setupNavigation();
    setupForms();
    setupRentalModal();
    await loadBikeTypes();
    await refreshAll();
    // Überfällig-Status und laufende Preise regelmäßig aktualisieren
    setInterval(refreshAll, 60000);
});

/* ---------- Navigation ---------- */

function setupNavigation() {
    document.querySelectorAll("[data-nav]").forEach((el) => {
        el.addEventListener("click", (e) => {
            e.preventDefault();
            switchView(el.getAttribute("data-nav"));
        });
    });
    const initial = location.hash.replace("#", "");
    if (initial) switchView(initial);
}

function switchView(viewName) {
    const target = document.getElementById(`view-${viewName}`);
    if (!target) return;
    document.querySelectorAll(".view").forEach((v) => v.classList.remove("active-view"));
    target.classList.add("active-view");
    document.querySelectorAll(".nav-link").forEach((link) => {
        link.classList.toggle("active", link.getAttribute("data-nav") === viewName);
    });
    history.replaceState(null, "", `#${viewName}`);
}

/* ---------- API ---------- */

async function apiRequest(url, options = {}) {
    const res = await fetch(url, { headers: { "Content-Type": "application/json" }, ...options });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.error || `HTTP-Fehler ${res.status}`);
    return data;
}

async function loadBikeTypes() {
    try {
        state.bikeTypes = await apiRequest("/api/bike-types");
    } catch (err) {
        showToast("Fahrrad-Typen konnten nicht geladen werden: " + err.message, true);
        return;
    }
    const types = Object.keys(state.bikeTypes).sort();
    const typeSelect = document.getElementById("bike-type");
    typeSelect.innerHTML = types
        .map((t) => `<option value="${escapeHtml(t)}">${escapeHtml(t)} (${eur.format(state.bikeTypes[t])}/Std.)</option>`)
        .join("");
    document.getElementById("filter-bike-type").innerHTML +=
        types.map((t) => `<option value="${escapeHtml(t)}">${escapeHtml(t)}</option>`).join("");
    syncRateWithType();
}

async function refreshAll() {
    try {
        const [stats, bikes, persons, reservations] = await Promise.all([
            apiRequest("/api/dashboard"),
            apiRequest("/api/bikes"),
            apiRequest("/api/persons"),
            apiRequest("/api/reservations"),
        ]);
        Object.assign(state, { stats, bikes, persons, reservations });
        renderDashboard();
        renderBikes();
        renderPersons();
        renderReservations();
        suggestNextBikeNumber();
    } catch (err) {
        showToast("Fehler beim Laden der Serverdaten: " + err.message, true);
    }
}

/* ---------- Formulare ---------- */

function setupForms() {
    document.getElementById("bike-type").addEventListener("change", syncRateWithType);

    document.getElementById("form-add-bike").addEventListener("submit", async (e) => {
        e.preventDefault();
        const number = document.getElementById("bike-number").value.trim();
        const type = document.getElementById("bike-type").value;
        const frameSizeCm = parseInt(document.getElementById("bike-frame").value, 10);
        const hourlyRate = parseFloat(document.getElementById("bike-rate").value);
        try {
            const bike = await apiRequest("/api/bikes", {
                method: "POST",
                body: JSON.stringify({ number, type, frameSizeCm, hourlyRate }),
            });
            document.getElementById("bike-number").value = "";
            showToast(`Fahrrad ${bike.number} (${bike.type}) angelegt.`);
            await refreshAll();
        } catch (err) {
            showToast(err.message, true);
        }
    });

    document.getElementById("form-add-person").addEventListener("submit", async (e) => {
        e.preventDefault();
        const name = document.getElementById("person-name").value.trim();
        const email = document.getElementById("person-email").value.trim();
        const phone = document.getElementById("person-phone").value.trim();
        try {
            await apiRequest("/api/persons", { method: "POST", body: JSON.stringify({ name, email, phone }) });
            e.target.reset();
            showToast(`Person "${name}" wurde registriert.`);
            await refreshAll();
        } catch (err) {
            showToast(err.message, true);
        }
    });

    document.getElementById("filter-bike-search").addEventListener("input", renderBikes);
    document.getElementById("filter-bike-type").addEventListener("change", renderBikes);
    document.getElementById("filter-bike-status").addEventListener("change", renderBikes);
    document.getElementById("filter-res-search").addEventListener("input", renderReservations);
    document.getElementById("filter-res-status").addEventListener("change", renderReservations);
}

function syncRateWithType() {
    const type = document.getElementById("bike-type").value;
    if (state.bikeTypes[type] !== undefined) {
        document.getElementById("bike-rate").value = state.bikeTypes[type];
    }
}

// Schlägt die nächste freie Inventarnummer vor (z.B. BIKE-121).
function suggestNextBikeNumber() {
    const input = document.getElementById("bike-number");
    const max = state.bikes.reduce((m, b) => {
        const n = parseInt((b.number.match(/(\d+)$/) || [])[1], 10);
        return Number.isFinite(n) && n > m ? n : m;
    }, 100);
    input.placeholder = `z.B. BIKE-${max + 1}`;
}

/* ---------- Ausleih-Modal ---------- */

function setupRentalModal() {
    const modal = document.getElementById("modal-rental");
    ["btn-open-rental-modal", "btn-open-rental-modal-2"].forEach((id) =>
        document.getElementById(id).addEventListener("click", () => openRentalModal())
    );
    const close = () => modal.classList.add("hidden");
    document.getElementById("btn-close-modal").addEventListener("click", close);
    document.getElementById("btn-cancel-modal").addEventListener("click", close);
    modal.addEventListener("click", (e) => { if (e.target === modal) close(); });
    document.addEventListener("keydown", (e) => { if (e.key === "Escape") close(); });

    document.querySelectorAll('input[name="rental-start-mode"]').forEach((r) =>
        r.addEventListener("change", () => {
            const later = getStartMode() === "later";
            document.getElementById("rental-start").classList.toggle("hidden", !later);
            loadAvailableBikes();
        })
    );
    document.getElementById("rental-start").addEventListener("change", loadAvailableBikes);
    document.getElementById("rental-duration").addEventListener("input", debounce(loadAvailableBikes, 250));
    document.getElementById("rental-unit").addEventListener("change", loadAvailableBikes);
    document.getElementById("rental-bike").addEventListener("change", updatePricePreview);

    document.querySelectorAll(".chip").forEach((chip) =>
        chip.addEventListener("click", () => {
            const h = parseInt(chip.dataset.hours, 10);
            const unit = h % 24 === 0 ? 24 : 1;
            document.getElementById("rental-unit").value = String(unit);
            document.getElementById("rental-duration").value = h / unit;
            loadAvailableBikes();
        })
    );

    document.getElementById("form-new-rental").addEventListener("submit", async (e) => {
        e.preventDefault();
        const personId = parseInt(document.getElementById("rental-person").value, 10);
        const bikeId = parseInt(document.getElementById("rental-bike").value, 10);
        const notes = document.getElementById("rental-notes").value.trim();
        const start = getStartISO();
        const hours = getHours();
        if (!bikeId) {
            showToast("Bitte ein freies Fahrrad auswählen.", true);
            return;
        }
        try {
            const r = await apiRequest("/api/reservations", {
                method: "POST",
                body: JSON.stringify({ bikeId, personId, start, hours, notes }),
            });
            close();
            document.getElementById("rental-notes").value = "";
            const what = r.status === "Aktiv" ? "ausgegeben" : `reserviert ab ${formatDateTime(r.plannedStart)}`;
            showToast(`#${r.id}: ${r.bikeNumber} an ${r.personName} ${what} (${eur.format(r.price)}).`);
            await refreshAll();
        } catch (err) {
            showToast(err.message, true);
        }
    });
}

function openRentalModal(preselectBikeId) {
    if (state.persons.length === 0) {
        showToast("Bitte zuerst mindestens eine Person registrieren!", true);
        return;
    }
    document.getElementById("rental-person").innerHTML = state.persons
        .map((p) => `<option value="${p.id}">${escapeHtml(p.name)} (${escapeHtml(p.email)})</option>`)
        .join("");

    // Standard für "Später": nächste volle Stunde
    const next = new Date();
    next.setMinutes(0, 0, 0);
    next.setHours(next.getHours() + 1);
    const startInput = document.getElementById("rental-start");
    startInput.value = toLocalInputValue(next);
    startInput.min = toLocalInputValue(new Date());

    document.getElementById("modal-rental").classList.remove("hidden");
    loadAvailableBikes(preselectBikeId);
}

function getStartMode() {
    return document.querySelector('input[name="rental-start-mode"]:checked').value;
}

function getStartISO() {
    if (getStartMode() === "now") return "";
    const v = document.getElementById("rental-start").value;
    return v ? new Date(v).toISOString() : "";
}

function getHours() {
    const n = parseInt(document.getElementById("rental-duration").value, 10) || 0;
    const unit = parseInt(document.getElementById("rental-unit").value, 10);
    return n * unit;
}

async function loadAvailableBikes(preselectBikeId) {
    const select = document.getElementById("rental-bike");
    const hint = document.getElementById("rental-bike-hint");
    const hours = getHours();
    if (hours < 1) {
        select.innerHTML = "";
        hint.textContent = "Bitte eine Dauer von mindestens 1 Stunde angeben.";
        updatePricePreview();
        return;
    }
    const params = new URLSearchParams({ hours: String(hours) });
    const start = getStartISO();
    if (start) params.set("start", start);
    try {
        const bikes = await apiRequest(`/api/bikes/available?${params}`);
        const previous = typeof preselectBikeId === "number" ? preselectBikeId : parseInt(select.value, 10);
        select.innerHTML = bikes
            .map((b) => `<option value="${b.id}" data-rate="${b.hourlyRate}">${escapeHtml(b.number)} – ${escapeHtml(b.type)}, ${b.frameSizeCm} cm · ${eur.format(b.hourlyRate)}/Std.</option>`)
            .join("");
        if (bikes.some((b) => b.id === previous)) select.value = String(previous);
        hint.textContent = bikes.length
            ? `${bikes.length} von ${state.bikes.length} Fahrrädern im Zeitraum frei.`
            : "Kein Fahrrad im gewählten Zeitraum frei – bitte Beginn oder Dauer ändern.";
        hint.classList.toggle("hint-error", bikes.length === 0);
    } catch (err) {
        select.innerHTML = "";
        hint.textContent = err.message;
        hint.classList.add("hint-error");
    }
    updatePricePreview();
}

function updatePricePreview() {
    const box = document.getElementById("price-preview");
    const opt = document.getElementById("rental-bike").selectedOptions[0];
    const hours = getHours();
    if (!opt || hours < 1) {
        box.innerHTML = "Preisvorschau: –";
        return;
    }
    const rate = parseFloat(opt.dataset.rate);
    const startDate = getStartISO() ? new Date(getStartISO()) : new Date();
    const end = new Date(startDate.getTime() + hours * 3600000);
    box.innerHTML = `
        <div><span>${hours} Std. × ${eur.format(rate)}</span><strong>${eur.format(rate * hours)}</strong></div>
        <small>Rückgabe geplant: ${formatDateTime(end.toISOString())} · abgerechnet wird jede angefangene Stunde der tatsächlichen Nutzung</small>`;
}

/* ---------- Rendering ---------- */

function renderDashboard() {
    const s = state.stats;
    setText("stat-available", s.availableBikes);
    setText("stat-rented", s.rentedBikes);
    setText("stat-total", s.totalBikes);
    setText("stat-persons", s.registeredPersons);
    setText("stat-upcoming", s.upcomingReserved);
    setText("stat-overdue", s.overdueRentals);
    setText("stat-revenue", eur.format(s.revenue || 0));
    document.querySelector(".mini-warn").classList.toggle("is-alert", s.overdueRentals > 0);

    const active = state.reservations.filter((r) => r.status === "Aktiv");
    document.getElementById("dashboard-active-body").innerHTML = active.length
        ? active.map((r) => `
            <tr class="${isOverdue(r) ? "row-overdue" : ""}">
                <td><strong>#${r.id}</strong></td>
                <td>${bikeCell(r)}</td>
                <td>${escapeHtml(r.personName)}</td>
                <td>${formatDateTime(r.pickupTime)}</td>
                <td>${formatDateTime(plannedEnd(r))} ${isOverdue(r) ? '<span class="badge badge-rented">überfällig</span>' : ""}</td>
                <td>${eur.format(runningPrice(r))}</td>
                <td><button class="btn-return-sm" onclick="reservationAction(${r.id}, 'return')">Rückgabe</button></td>
            </tr>`).join("")
        : emptyRow(7, "Derzeit sind keine Fahrräder ausgeliehen.");

    const upcoming = state.reservations.filter((r) => r.status === "Reserviert").slice(0, 5);
    document.getElementById("dashboard-upcoming-body").innerHTML = upcoming.length
        ? upcoming.map((r) => `
            <tr>
                <td><strong>#${r.id}</strong></td>
                <td>${bikeCell(r)}</td>
                <td>${escapeHtml(r.personName)}</td>
                <td>${formatDateTime(r.plannedStart)}</td>
                <td>${formatHours(r.plannedHours)}</td>
                <td>${eur.format(r.price)}</td>
                <td class="actions">
                    <button class="btn-return-sm" onclick="reservationAction(${r.id}, 'pickup')">Abholen</button>
                    <button class="btn-danger-sm" onclick="reservationAction(${r.id}, 'cancel')">Storno</button>
                </td>
            </tr>`).join("")
        : emptyRow(7, "Keine geplanten Reservierungen.");
}

function renderBikes() {
    const search = document.getElementById("filter-bike-search").value.trim().toLowerCase();
    const typeFilter = document.getElementById("filter-bike-type").value;
    const statusFilter = document.getElementById("filter-bike-status").value;

    const activeByBike = {};
    const nextByBike = {};
    state.reservations.forEach((r) => {
        if (r.status === "Aktiv") activeByBike[r.bikeId] = r;
        if (r.status === "Reserviert" && !nextByBike[r.bikeId]) nextByBike[r.bikeId] = r; // Liste ist nach Beginn sortiert
    });

    const filtered = state.bikes.filter((b) => {
        if (statusFilter === "available" && !b.available) return false;
        if (statusFilter === "rented" && b.available) return false;
        if (typeFilter !== "all" && b.type !== typeFilter) return false;
        if (search && !`${b.number} ${b.type} ${b.frameSizeCm}`.toLowerCase().includes(search)) return false;
        return true;
    });

    const tbody = document.getElementById("bikes-table-body");
    if (filtered.length === 0) {
        tbody.innerHTML = emptyRow(8, "Keine passenden Fahrräder gefunden.");
        return;
    }
    tbody.innerHTML = filtered.map((b) => {
        const act = activeByBike[b.id];
        const next = nextByBike[b.id];
        const status = b.available
            ? `<span class="badge badge-available">Verfügbar</span>`
            : `<span class="badge badge-rented">Ausgeliehen</span>`;
        let info = "–";
        if (act) {
            info = `${escapeHtml(act.personName)} <small class="muted">bis ${formatDateTime(plannedEnd(act))}</small>`;
        } else if (next) {
            info = `<small class="muted">reserviert ab ${formatDateTime(next.plannedStart)} (${escapeHtml(next.personName)})</small>`;
        }
        const actions = act
            ? `<button class="btn-return-sm" onclick="reservationAction(${act.id}, 'return')">Rückgabe</button>`
            : `<button class="btn-primary-sm" onclick="openRentalModal(${b.id})">Ausleihen</button>
               <button class="btn-danger-sm" onclick="deleteBike(${b.id})">Löschen</button>`;
        return `
            <tr>
                <td>${b.id}</td>
                <td><strong>${escapeHtml(b.number)}</strong></td>
                <td><span class="badge badge-type">${escapeHtml(b.type)}</span></td>
                <td>${b.frameSizeCm} cm</td>
                <td>${eur.format(b.hourlyRate)}</td>
                <td>${status}</td>
                <td>${info}</td>
                <td class="actions">${actions}</td>
            </tr>`;
    }).join("");
}

function renderPersons() {
    const openCount = {};
    state.reservations.forEach((r) => {
        if (r.status === "Aktiv" || r.status === "Reserviert") openCount[r.personId] = (openCount[r.personId] || 0) + 1;
    });
    const tbody = document.getElementById("persons-table-body");
    if (state.persons.length === 0) {
        tbody.innerHTML = emptyRow(6, "Keine Personen registriert.");
        return;
    }
    tbody.innerHTML = state.persons.map((p) => {
        const c = openCount[p.id] || 0;
        return `
            <tr>
                <td>${p.id}</td>
                <td><strong>${escapeHtml(p.name)}</strong></td>
                <td>${escapeHtml(p.email)}</td>
                <td>${escapeHtml(p.phone || "–")}</td>
                <td><span class="badge ${c ? "badge-rented" : "badge-returned"}">${c} offen</span></td>
                <td><button class="btn-danger-sm" onclick="deletePerson(${p.id})">Löschen</button></td>
            </tr>`;
    }).join("");
}

function renderReservations() {
    const search = document.getElementById("filter-res-search").value.trim().toLowerCase();
    const statusFilter = document.getElementById("filter-res-status").value;
    const filtered = state.reservations.filter((r) => {
        if (statusFilter !== "all" && r.status !== statusFilter) return false;
        if (search && !`${r.bikeNumber} ${r.bikeType} ${r.personName} ${r.notes || ""}`.toLowerCase().includes(search)) return false;
        return true;
    });

    const tbody = document.getElementById("reservations-table-body");
    if (filtered.length === 0) {
        tbody.innerHTML = emptyRow(10, "Keine Reservierungen für diesen Filter vorhanden.");
        return;
    }
    tbody.innerHTML = filtered.map((r) => {
        let action = "–";
        if (r.status === "Aktiv") {
            action = `<button class="btn-return-sm" onclick="reservationAction(${r.id}, 'return')">Rückgabe</button>`;
        } else if (r.status === "Reserviert") {
            action = `<button class="btn-return-sm" onclick="reservationAction(${r.id}, 'pickup')">Abholen</button>
                      <button class="btn-danger-sm" onclick="reservationAction(${r.id}, 'cancel')">Storno</button>`;
        }
        const begin = r.pickupTime ? formatDateTime(r.pickupTime) : `<em>${formatDateTime(r.plannedStart)}</em>`;
        let price = eur.format(r.price);
        if (r.status === "Aktiv") price = `${eur.format(runningPrice(r))} <small class="muted">bisher</small>`;
        if (r.status === "Reserviert") price = `${eur.format(r.price)} <small class="muted">geplant</small>`;
        return `
            <tr class="${isOverdue(r) ? "row-overdue" : ""}">
                <td><strong>#${r.id}</strong></td>
                <td>${bikeCell(r)}</td>
                <td>${escapeHtml(r.personName)}</td>
                <td>${begin}</td>
                <td>${formatDateTime(plannedEnd(r))} <small class="muted">(${formatHours(r.plannedHours)})</small></td>
                <td>${r.returnTime ? formatDateTime(r.returnTime) : "–"}</td>
                <td>${price}</td>
                <td>${statusBadge(r)}</td>
                <td>${escapeHtml(r.notes || "–")}</td>
                <td class="actions">${action}</td>
            </tr>`;
    }).join("");
}

/* ---------- Aktionen ---------- */

async function reservationAction(id, action) {
    if (action === "cancel" && !confirm(`Reservierung #${id} wirklich stornieren?`)) return;
    try {
        const r = await apiRequest(`/api/reservations/${id}/${action}`, { method: "POST" });
        const msg = {
            pickup: `${r.bikeNumber} an ${r.personName} ausgegeben.`,
            return: `${r.bikeNumber} zurückgegeben – Endpreis ${eur.format(r.price)}.`,
            cancel: `Reservierung #${r.id} storniert.`,
        }[action];
        showToast(msg);
        await refreshAll();
    } catch (err) {
        showToast(err.message, true);
    }
}

async function deleteBike(id) {
    if (!confirm(`Fahrrad #${id} wirklich löschen?`)) return;
    try {
        await apiRequest(`/api/bikes/${id}`, { method: "DELETE" });
        showToast(`Fahrrad #${id} wurde gelöscht.`);
        await refreshAll();
    } catch (err) {
        showToast(err.message, true);
    }
}

async function deletePerson(id) {
    if (!confirm(`Person #${id} wirklich löschen?`)) return;
    try {
        await apiRequest(`/api/persons/${id}`, { method: "DELETE" });
        showToast(`Person #${id} wurde gelöscht.`);
        await refreshAll();
    } catch (err) {
        showToast(err.message, true);
    }
}

/* ---------- Helfer ---------- */

function plannedEnd(r) {
    return new Date(new Date(r.plannedStart).getTime() + r.plannedHours * 3600000).toISOString();
}

function isOverdue(r) {
    return r.status === "Aktiv" && Date.now() > new Date(plannedEnd(r)).getTime();
}

// Laufender Preis einer aktiven Ausleihe (jede angefangene Stunde, mind. 1).
function runningPrice(r) {
    const start = new Date(r.pickupTime || r.plannedStart).getTime();
    const hours = Math.max(1, Math.ceil((Date.now() - start) / 3600000));
    return Math.round(hours * r.hourlyRate * 100) / 100;
}

function bikeCell(r) {
    return `<strong>${escapeHtml(r.bikeNumber)}</strong> <span class="badge badge-type">${escapeHtml(r.bikeType)} · ${r.bikeFrameCm} cm</span>`;
}

function statusBadge(r) {
    if (isOverdue(r)) return `<span class="badge badge-rented">Überfällig</span>`;
    const cls = {
        "Aktiv": "badge-rented",
        "Reserviert": "badge-reserved",
        "Zurückgegeben": "badge-returned",
        "Storniert": "badge-cancelled",
    }[r.status] || "badge-returned";
    return `<span class="badge ${cls}">${escapeHtml(r.status)}</span>`;
}

function formatHours(h) {
    if (h % 24 === 0) return `${h / 24} ${h === 24 ? "Tag" : "Tage"}`;
    return `${h} Std.`;
}

function emptyRow(cols, text) {
    return `<tr><td colspan="${cols}" class="empty">${text}</td></tr>`;
}

function setText(id, v) {
    document.getElementById(id).textContent = v ?? "–";
}

function toLocalInputValue(d) {
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function formatDateTime(iso) {
    if (!iso) return "–";
    return new Date(iso).toLocaleString("de-AT", {
        day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit",
    });
}

function escapeHtml(str) {
    return String(str)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
}

function debounce(fn, ms) {
    let t;
    return (...args) => { clearTimeout(t); t = setTimeout(() => fn(...args), ms); };
}

let toastTimer = null;
function showToast(msg, isError = false) {
    const toast = document.getElementById("toast");
    toast.textContent = msg;
    toast.classList.toggle("error", isError);
    toast.classList.remove("hidden");
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => toast.classList.add("hidden"), 4000);
}
