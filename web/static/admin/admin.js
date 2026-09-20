(() => {
  "use strict";

  const TOKEN_KEY = "otojoga_admin_token";

  // must match internal/application/location/link_provider.go — an unknown
  // location breaks GET /api/v1/classes for every class, not just this one
  const LOCATIONS = ["ożarowska 75/36", "ogród krasińskich", "ogród saski", "park moczydło"];

  const state = {
    token: null,
    classes: [],
    bookings: [],
    pendingBookings: [],
    contacts: [],
    passes: [],
  };

  // ---------- dom helpers ----------

  function el(tag, attrs = {}, children = []) {
    const node = document.createElement(tag);
    for (const [key, value] of Object.entries(attrs)) {
      if (key === "class") node.className = value;
      else if (key.startsWith("on") && typeof value === "function") node.addEventListener(key.slice(2), value);
      else if (value !== null && value !== undefined) node.setAttribute(key, value);
    }
    for (const child of [].concat(children)) {
      if (child === null || child === undefined) continue;
      node.append(child instanceof Node ? child : document.createTextNode(String(child)));
    }
    return node;
  }

  function td(content) {
    return el("td", {}, [content]);
  }

  function clear(node) {
    node.replaceChildren();
  }

  function toast(message, kind = "good") {
    const root = document.getElementById("toast-root");
    const node = el("div", { class: `toast ${kind}` }, [message]);
    root.append(node);
    setTimeout(() => node.remove(), 4500);
  }

  function showModal({
    message,
    withInput = false,
    inputValue = "",
    confirmLabel = "ok",
    confirmClass = "btn-primary",
    cancelLabel = null,
  }) {
    return new Promise((resolve) => {
      const input = withInput ? el("input", { type: "text", value: inputValue }) : null;

      let resolved = false;
      const finish = (value) => {
        resolved = true;
        dialog.close();
        resolve(value);
      };

      const actions = el("div", { class: "dialog-actions" }, [
        cancelLabel ? el("button", { class: "btn btn-sm", type: "button", onclick: () => finish(null) }, cancelLabel) : null,
        el("button", {
          class: `btn btn-sm ${confirmClass}`,
          type: "button",
          onclick: () => finish(withInput ? input.value : true),
        }, confirmLabel),
      ]);

      const dialog = el("dialog", { class: "dialog confirm-dialog" }, [
        el("p", {}, message),
        input,
        actions,
      ]);

      dialog.addEventListener("close", () => {
        dialog.remove();
        if (!resolved) resolve(null);
      });

      document.body.append(dialog);
      dialog.showModal();
      if (input) input.focus();
    });
  }

  // ---------- api ----------

  async function api(path, { method = "GET", body } = {}) {
    const headers = { Authorization: `Bearer ${state.token}` };
    if (body !== undefined) headers["Content-Type"] = "application/json";

    const res = await fetch(path, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });

    if (res.status === 401) {
      logout("Sesja wygasła — wpisz token ponownie.");
      throw new Error("unauthorized");
    }

    const text = await res.text();
    const data = text ? JSON.parse(text) : null;

    if (!res.ok) {
      throw new Error((data && data.error) || `Błąd ${res.status}`);
    }

    return data;
  }

  // ---------- auth gate ----------

  function logout(message) {
    state.token = null;
    localStorage.removeItem(TOKEN_KEY);
    document.getElementById("app").hidden = true;
    document.getElementById("gate").hidden = false;
    const errorNode = document.getElementById("gate-error");
    if (message) {
      errorNode.textContent = message;
      errorNode.hidden = false;
    }
  }

  async function tryEnter(token) {
    state.token = token;
    try {
      await api("/api/v1/contacts");
    } catch (err) {
      state.token = null;
      throw err;
    }
    localStorage.setItem(TOKEN_KEY, token);
    document.getElementById("gate").hidden = true;
    document.getElementById("app").hidden = false;
    loadAll();
  }

  // ---------- classes ----------

  function parseClassDate(cls) {
    const [day, month, year] = cls.start_date.split("-").map(Number);
    const [hour, minute] = cls.start_hour.split(":").map(Number);
    return new Date(year, month - 1, day, hour, minute);
  }

  function formatClassDateTime(cls) {
    return parseClassDate(cls).toLocaleString("pl-PL");
  }

  function toDateTimeLocalValue(date) {
    const pad = (n) => String(n).padStart(2, "0");
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }

  async function loadClasses() {
    state.classes = await api("/api/v1/classes");
    renderClasses();
  }

  function renderClasses() {
    const tbody = document.querySelector("#classes-table tbody");
    const emptyNode = document.querySelector("#classes .empty");
    clear(tbody);

    const onlyUpcoming = document.getElementById("classes-upcoming").checked;
    const now = new Date();

    const rows = state.classes
      .map((cls) => ({ cls, date: parseClassDate(cls) }))
      .filter((row) => !onlyUpcoming || row.date >= now)
      .sort((a, b) => a.date - b.date);

    emptyNode.hidden = rows.length > 0;

    for (const { cls } of rows) {
      tbody.append(renderClassRow(cls));
    }
  }

  function renderClassRow(cls) {
    const row = el("tr", { "data-id": cls.id });

    const bookedCount = cls.max_capacity - cls.current_capacity;

    row.append(
      td(formatClassDateTime(cls)),
      td(cls.class_name),
      td(cls.class_level),
      td(cls.location),
      td(`${bookedCount} / ${cls.max_capacity}`),
    );

    const actions = el("div", { class: "btn-row" }, [
      el("button", { class: "btn btn-sm", type: "button", onclick: () => startEditClass(cls, row) }, "Edytuj"),
      el("button", { class: "btn btn-sm", type: "button", onclick: () => showClassBookings(cls) }, "Rezerwacje"),
      el("button", { class: "btn btn-sm btn-danger", type: "button", onclick: () => deleteClass(cls) }, "Usuń"),
    ]);
    row.append(el("td", {}, [actions]));

    return row;
  }

  function startEditClass(cls, row) {
    clear(row);
    row.dataset.id = cls.id;

    const startInput = el("input", { type: "datetime-local", value: toDateTimeLocalValue(parseClassDate(cls)) });
    const nameInput = el("input", { type: "text", value: cls.class_name });
    const levelInput = el("input", { type: "text", value: cls.class_level });
    const locationInput = el("select", {}, LOCATIONS.map((loc) =>
      el("option", { value: loc, selected: loc === cls.location ? "" : null }, loc),
    ));
    const capacityInput = el("input", { type: "number", min: "1", value: cls.max_capacity, class: "cap-input" });

    const messageInput = el("input", {
      type: "text",
      class: "msg-input-wide",
      placeholder: "wiadomość dla zapisanych (wymagana przy zmianie terminu/miejsca)",
    });

    row.append(
      el("td", { class: "edit-term-cell" }, [el("div", { class: "cell-stack" }, [startInput, messageInput])]),
      td(nameInput),
      td(levelInput),
      td(locationInput),
      td(capacityInput),
    );

    messageInput.style.width = `${capacityInput.getBoundingClientRect().right - messageInput.parentElement.getBoundingClientRect().left}px`;

    const actions = el("div", { class: "btn-row" }, [
      el("button", {
        class: "btn btn-sm btn-primary",
        type: "button",
        onclick: () => {
          if (!messageInput.value.trim()) {
            toast("Podaj wiadomość dla zapisanych osób — zmiana terminu/miejsca tego wymaga.", "crit");
            return;
          }
          saveClass(cls, {
            start_time: `${startInput.value}:00`,
            class_name: nameInput.value,
            class_level: levelInput.value,
            location: locationInput.value,
            max_capacity: Number(capacityInput.value),
            message: messageInput.value.trim(),
          });
        },
      }, "Zapisz"),
      el("button", { class: "btn btn-sm", type: "button", onclick: renderClasses }, "Anuluj"),
    ]);
    row.append(el("td", {}, [actions]));
  }

  async function saveClass(cls, patch) {
    try {
      await api(`/api/v1/classes/${cls.id}`, { method: "PATCH", body: patch });
      toast("Zajęcia zaktualizowane.");
      await loadClasses();
    } catch (err) {
      toast(err.message, "crit");
    }
  }

  async function deleteClass(cls) {
    const confirmed = await showModal({
      message: [
        "Na pewno usunąć zajęcia?", el("br"),
        el("em", {}, cls.class_name), el("br"),
        el("span", { class: "dim" }, formatClassDateTime(cls)),
      ],
      confirmLabel: "usuń",
      cancelLabel: "anuluj",
    });
    if (!confirmed) return;

    const message = await showModal({
      message: "Wiadomość dla zapisanych osób (opcjonalnie):",
      withInput: true,
      confirmLabel: "ok",
      cancelLabel: "pomiń",
    });

    try {
      await api(`/api/v1/classes/${cls.id}`, { method: "DELETE", body: { message: message || null } });
      toast("Zajęcia usunięte.");
      await loadClasses();
    } catch (err) {
      toast(err.message, "crit");
    }
  }

  async function showClassBookings(cls) {
    try {
      const bookings = await api(`/api/v1/classes/${cls.id}/bookings`);
      if (bookings.length === 0) {
        toast(`Brak rezerwacji na „${cls.class_name}”.`);
        return;
      }
      const names = bookings.map((b) => `${b.first_name} ${b.last_name}`).join(", ");
      await showModal({
        message: [
          el("em", {}, cls.class_name),
          el("br"),
          el("span", { class: "dim" }, formatClassDateTime(cls)),
          el("br"), el("br"),
          names,
        ],
        confirmClass: "btn-accent",
      });
    } catch (err) {
      toast(err.message, "crit");
    }
  }

  async function createClass(form) {
    const data = new FormData(form);
    const payload = [{
      start_time: `${data.get("start_time")}:00`,
      class_name: data.get("class_name"),
      class_level: data.get("class_level"),
      location: data.get("location"),
      max_capacity: Number(data.get("max_capacity")),
    }];

    try {
      await api("/api/v1/classes", { method: "POST", body: payload });
      toast("Zajęcia utworzone.");
      form.reset();
      form.closest("dialog").close();
      await loadClasses();
    } catch (err) {
      toast(err.message, "crit");
    }
  }

  // ---------- bookings ----------

  async function loadBookings() {
    state.bookings = await api("/api/v1/bookings");
    renderBookings();
  }

  function renderBookings() {
    const tbody = document.querySelector("#bookings-table tbody");
    const emptyNode = document.querySelector("#bookings .empty");
    clear(tbody);

    emptyNode.hidden = state.bookings.length > 0;

    for (const booking of state.bookings) {
      tbody.append(renderBookingRow(booking));
    }
  }

  function renderBookingRow(booking) {
    const row = el("tr", { "data-id": booking.id });

    row.append(
      td(`${booking.first_name} ${booking.last_name}`),
      td(booking.email),
      td(booking.class.class_name),
      td(formatClassDateTime(booking.class)),
      td(booking.pass
        ? el("span", { class: "badge" }, `#${booking.pass.id}`)
        : el("span", { class: "badge none" }, "brak")),
    );

    row.append(el("td", {}, [
      el("button", {
        class: "btn btn-sm btn-danger",
        type: "button",
        onclick: () => deleteBooking(booking),
      }, "Anuluj"),
    ]));

    return row;
  }

  async function deleteBooking(booking) {
    const confirmed = await showModal({
      message: [
        "Na pewno anulować rezerwację?", el("br"),
        el("em", {}, `${booking.first_name} ${booking.last_name}`), el("br"),
        el("span", { class: "dim" }, formatClassDateTime(booking.class)),
      ],
      confirmLabel: "anuluj",
      cancelLabel: "wróć",
    });
    if (!confirmed) return;

    try {
      await api(`/api/v1/bookings/${booking.id}`, { method: "DELETE" });
      toast("Rezerwacja anulowana.");
      await loadBookings();
    } catch (err) {
      toast(err.message, "crit");
    }
  }

  // ---------- pending bookings ----------

  async function loadPendingBookings() {
    state.pendingBookings = await api("/api/v1/pending_bookings");
    renderPendingBookings();
  }

  function renderPendingBookings() {
    const tbody = document.querySelector("#pending-table tbody");
    const emptyNode = document.querySelector("#pending .empty");
    clear(tbody);

    emptyNode.hidden = state.pendingBookings.length > 0;

    for (const pending of state.pendingBookings) {
      const row = el("tr");
      row.append(
        td(`${pending.first_name} ${pending.last_name}`),
        td(pending.email),
        td(pending.class ? pending.class.ClassName : ""),
        td(pending.class ? new Date(pending.class.StartTime).toLocaleString("pl-PL") : ""),
      );
      tbody.append(row);
    }
  }

  // ---------- passes ----------

  async function activatePass(form) {
    const data = new FormData(form);
    const payload = {
      email: data.get("email"),
      initial_assigned_slots: Number(data.get("initial_assigned_slots")),
      total_slots: Number(data.get("total_slots")),
    };

    try {
      const result = await api("/api/v1/passes", { method: "PUT", body: payload });
      toast("Karnet aktywowany.");
      renderPassResult(result);
      form.reset();
      await loadPasses();
    } catch (err) {
      toast(err.message, "crit");
    }
  }

  function renderPassResult(result) {
    const node = document.getElementById("pass-result");
    clear(node);
    node.hidden = false;

    node.append(el("h3", {}, "Wynik aktywacji"));
    node.append(el("p", {}, [
      `Karnet #${result.pass.id} — `,
      result.pass.email,
      ` — ${result.pass.total_slots} slotów`,
    ]));

    if (result.updated_bookings.length > 0) {
      const shown = result.updated_bookings.slice(0, 3);
      const list = el("ul", {}, shown.map((b) =>
        el("li", {}, `${b.class.class_name} — ${formatClassDateTime(b.class)}`),
      ));
      const remaining = result.updated_bookings.length - shown.length;
      node.append(el("p", {}, "Zaktualizowane rezerwacje:"), list);
      if (remaining > 0) node.append(el("p", { class: "dim" }, `+ ${remaining} więcej`));
    }
  }

  async function loadPasses() {
    state.passes = await api("/api/v1/passes");
    renderPasses();
  }

  function renderPasses() {
    const tbody = document.querySelector("#passes-table tbody");
    const emptyNode = document.querySelector("#passes-list .empty");
    clear(tbody);

    emptyNode.hidden = state.passes.length > 0;

    for (const pass of state.passes) {
      const row = el("tr");
      row.append(
        td(String(pass.id)),
        td(pass.email),
        td(`${pass.used_slots} / ${pass.total_slots}`),
        td(el("span", { class: "dim" }, new Date(pass.created_at).toLocaleString("pl-PL"))),
      );
      tbody.append(row);
    }
  }

  // ---------- contacts ----------

  async function loadContacts() {
    state.contacts = await api("/api/v1/contacts");
    renderContacts();
  }

  function renderContacts() {
    const tbody = document.querySelector("#contacts-table tbody");
    const emptyNode = document.querySelector("#contacts .empty");
    clear(tbody);

    emptyNode.hidden = state.contacts.length > 0;

    for (const contact of state.contacts) {
      const row = el("tr");
      row.append(td(String(contact.id)), td(contact.email), td(contact.first_name), td(contact.last_name));
      tbody.append(row);
    }
  }

  async function createContact(form) {
    const data = new FormData(form);
    const payload = [{
      email: data.get("email"),
      first_name: data.get("first_name"),
      last_name: data.get("last_name"),
    }];

    try {
      await api("/api/v1/contacts", { method: "POST", body: payload });
      toast("Kontakt dodany.");
      form.reset();
      form.closest("dialog").close();
      await loadContacts();
    } catch (err) {
      toast(err.message, "crit");
    }
  }

  // ---------- bootstrap ----------

  async function loadAll() {
    try {
      await Promise.all([loadClasses(), loadBookings(), loadPendingBookings(), loadContacts(), loadPasses()]);
    } catch (err) {
      if (err.message !== "unauthorized") toast(err.message, "crit");
    }
  }

  function wireDialogs() {
    document.querySelectorAll("[data-open]").forEach((trigger) => {
      trigger.addEventListener("click", () => {
        document.getElementById(trigger.dataset.open).showModal();
      });
    });

    document.querySelectorAll("dialog").forEach((dialog) => {
      dialog.querySelectorAll("[data-close]").forEach((btn) => {
        btn.addEventListener("click", () => dialog.close());
      });
      dialog.addEventListener("close", () => dialog.querySelector("form")?.reset());
    });
  }

  function wireEvents() {
    wireDialogs();

    document.getElementById("gate-form").addEventListener("submit", async (e) => {
      e.preventDefault();
      const tokenInput = document.getElementById("gate-token");
      const errorNode = document.getElementById("gate-error");
      errorNode.hidden = true;

      try {
        await tryEnter(tokenInput.value.trim());
      } catch {
        errorNode.textContent = "Nieprawidłowy token.";
        errorNode.hidden = false;
      }
    });

    document.getElementById("logout").addEventListener("click", () => logout());

    document.getElementById("classes-upcoming").addEventListener("change", renderClasses);
    document.getElementById("class-create-form").addEventListener("submit", (e) => {
      e.preventDefault();
      createClass(e.target);
    });

    document.getElementById("pass-form").addEventListener("submit", (e) => {
      e.preventDefault();
      activatePass(e.target);
    });

    document.getElementById("contact-create-form").addEventListener("submit", (e) => {
      e.preventDefault();
      createContact(e.target);
    });
  }

  function init() {
    wireEvents();

    const savedToken = localStorage.getItem(TOKEN_KEY);
    if (!savedToken) return;

    tryEnter(savedToken).catch(() => {
      document.getElementById("gate-error").hidden = true;
    });
  }

  document.addEventListener("DOMContentLoaded", init);
})();
