(() => {
  "use strict";

  const elements = {
    localTime: document.querySelector("#local-time"),
    localDate: document.querySelector("#local-date"),
    localZone: document.querySelector("#local-zone"),
    utcTime: document.querySelector("#utc-time"),
    utcDate: document.querySelector("#utc-date"),
    unixTime: document.querySelector("#unix-time"),
    syncLabel: document.querySelector("#sync-label"),
    syncState: document.querySelector(".sync-state"),
    monitorState: document.querySelector("#monitor-state"),
    requestTotal: document.querySelector("#request-total"),
    clientTotal: document.querySelector("#client-total"),
    ntpStratum: document.querySelector("#ntp-stratum"),
    requestRows: document.querySelector("#request-rows"),
    retentionNote: document.querySelector("#retention-note"),
  };

  let serverOffset = 0;
  let lastRequestID = -1;
  let requestUpdateRunning = false;

  const localTimeFormatter = new Intl.DateTimeFormat("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hourCycle: "h23",
  });

  const localDateFormatter = new Intl.DateTimeFormat("en-GB", {
    weekday: "long",
    day: "2-digit",
    month: "long",
    year: "numeric",
  });

  const twoDigits = (value) => String(value).padStart(2, "0");

  function render() {
    const now = new Date(Date.now() + serverOffset);
    const utcTime = `${twoDigits(now.getUTCHours())}:${twoDigits(now.getUTCMinutes())}:${twoDigits(now.getUTCSeconds())}`;
    const utcDate = `${now.getUTCFullYear()}-${twoDigits(now.getUTCMonth() + 1)}-${twoDigits(now.getUTCDate())}`;

    elements.localTime.textContent = localTimeFormatter.format(now);
    elements.localTime.dateTime = now.toISOString();
    elements.localDate.textContent = localDateFormatter.format(now);
    elements.utcTime.textContent = utcTime;
    elements.utcTime.dateTime = now.toISOString();
    elements.utcDate.textContent = utcDate;
    elements.unixTime.textContent = Math.floor(now.getTime() / 1000).toString();
  }

  async function synchronize() {
    const requestStarted = Date.now();

    try {
      const response = await fetch("/api/time", { cache: "no-store" });
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const data = await response.json();
      const requestFinished = Date.now();
      const estimatedClientTime = requestStarted + (requestFinished - requestStarted) / 2;
      serverOffset = data.unixMilliseconds - estimatedClientTime;
      elements.syncLabel.textContent = "Server synchronized";
      elements.syncState.classList.remove("is-local");
    } catch {
      serverOffset = 0;
      elements.syncLabel.textContent = "Local system time";
      elements.syncState.classList.add("is-local");
    }

    render();
  }

  function requestTime(timestamp) {
    const date = new Date(timestamp);
    const milliseconds = String(date.getMilliseconds()).padStart(3, "0");
    return `${localTimeFormatter.format(date)}.${milliseconds}`;
  }

  function tableCell(text, className = "") {
    const cell = document.createElement("td");
    cell.textContent = text;
    if (className) {
      cell.className = className;
    }
    return cell;
  }

  function renderNTPRequests(data) {
    elements.requestTotal.textContent = data.total.toLocaleString("en-GB");
    elements.ntpStratum.textContent = String(data.stratum);
    elements.retentionNote.textContent = `The latest ${data.capacity} requests are retained until the next service restart.`;

    const clients = new Set(data.requests.map((request) => request.clientIp));
    elements.clientTotal.textContent = clients.size.toLocaleString("en-GB");

    const newestID = data.requests[0]?.id ?? 0;
    if (newestID === lastRequestID) {
      return;
    }
    lastRequestID = newestID;

    if (data.requests.length === 0) {
      const row = document.createElement("tr");
      row.className = "empty-row";
      const cell = tableCell("No NTP requests received yet.");
      cell.colSpan = 5;
      row.append(cell);
      elements.requestRows.replaceChildren(row);
      return;
    }

    const rows = data.requests.map((request) => {
      const row = document.createElement("tr");
      const endpoint = request.clientPort > 0 ? `${request.clientIp}:${request.clientPort}` : request.clientIp;
      row.append(
        tableCell(requestTime(request.receivedAt), "request-time"),
        tableCell(endpoint, "request-client"),
        tableCell(`NTPv${request.version} / M${request.mode}`),
        tableCell(`${request.bytes} B`),
        tableCell(request.result, `request-result request-result--${request.result}`),
      );
      return row;
    });
    elements.requestRows.replaceChildren(...rows);
  }

  async function updateNTPRequests() {
    if (requestUpdateRunning) {
      return;
    }
    requestUpdateRunning = true;

    try {
      const response = await fetch("/api/ntp/requests", { cache: "no-store" });
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      renderNTPRequests(await response.json());
      elements.monitorState.classList.remove("is-offline");
      elements.monitorState.lastElementChild.textContent = "Live";
    } catch {
      elements.monitorState.classList.add("is-offline");
      elements.monitorState.lastElementChild.textContent = "Disconnected";
    } finally {
      requestUpdateRunning = false;
    }
  }

  try {
    elements.localZone.textContent = Intl.DateTimeFormat().resolvedOptions().timeZone || "Local time zone";
  } catch {
    elements.localZone.textContent = "Local time zone";
  }

  render();
  synchronize();
  updateNTPRequests();
  setInterval(render, 250);
  setInterval(updateNTPRequests, 1000);
  setInterval(synchronize, 5 * 60 * 1000);
})();
