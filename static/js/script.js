const highlight = () => {
  const highlightedRows = document.querySelectorAll(
    "#poop-log > table > tbody > tr.highlighted"
  );
  highlightedRows.forEach((e) => e.classList.remove("highlighted"));
  const hash = location.hash.substring(1);
  if (hash) {
    const row = document.getElementById(hash);
    if (row) {
      row.classList.add("highlighted");
    }
  }
};

const isIOS =
  /iPad|iPhone|iPod/.test(navigator.userAgent) ||
  (navigator.userAgent.includes("Mac") && navigator.maxTouchPoints > 1);

class SSEClient {
  constructor(url, options = {}) {
    this.url = url;
    this.onMessage = options.onMessage || (() => {});
    this.onError = options.onError || (() => {});
    this.eventSource = null;
    this.isConnected = false;
    this.isPaused = false;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 10;
    this.baseReconnectDelay = 1000;

    this._bindLifecycleHandlers();
  }

  connect() {
    if (this.eventSource || this.isPaused) {
      return;
    }

    try {
      this.eventSource = new EventSource(this.url);

      this.eventSource.addEventListener("poopupdate", (event) => {
        this.reconnectAttempts = 0;
        this.onMessage(event);
      });

      this.eventSource.addEventListener("open", () => {
        this.isConnected = true;
        this.reconnectAttempts = 0;
      });

      this.eventSource.addEventListener("error", (event) => {
        this.isConnected = false;
        this.onError(event);

        if (this.eventSource?.readyState === EventSource.CLOSED) {
          this._scheduleReconnect();
        }
      });
    } catch {
      this._scheduleReconnect();
    }
  }

  disconnect() {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
      this.isConnected = false;
    }
  }

  reconnect() {
    this.disconnect();
    this.isPaused = false;
    this.connect();
  }

  pause() {
    this.isPaused = true;
    this.disconnect();
  }

  resume() {
    this.isPaused = false;
    this.connect();
  }

  _scheduleReconnect() {
    if (this.isPaused || this.reconnectAttempts >= this.maxReconnectAttempts) {
      return;
    }

    const delay = Math.min(
      this.baseReconnectDelay * Math.pow(2, this.reconnectAttempts),
      30000
    );
    this.reconnectAttempts++;

    setTimeout(() => {
      if (!this.isPaused && !this.isConnected) {
        this.reconnect();
      }
    }, delay);
  }

  _bindLifecycleHandlers() {
    document.addEventListener("visibilitychange", () => {
      if (document.visibilityState === "visible") {
        if (!this.isPaused && !this.isConnected) {
          this.reconnect();
        }
      } else if (isIOS) {
        this.disconnect();
      }
    });

    if ("onfreeze" in document) {
      document.addEventListener("freeze", () => {
        this.disconnect();
      });

      document.addEventListener("resume", () => {
        if (!this.isPaused) {
          this.reconnect();
        }
      });
    }

    window.addEventListener("pagehide", (event) => {
      if (!event.persisted && !isIOS) {
        this.disconnect();
      }
    });

    window.addEventListener("pageshow", (event) => {
      if (event.persisted && !this.isPaused && !this.isConnected) {
        this.reconnect();
      }
    });
  }
}

let sseClient = null;

const listenToPoopEvent = (period, year, month, triggerHighlight = false) => {
  const param = new URLSearchParams();
  param.append("period", period);
  param.append("year", year);
  if (month) {
    param.append("month", month);
  }

  sseClient = new SSEClient(`/sse?${param.toString()}`, {
    onMessage: (event) => {
      const data = JSON.parse(event.data);
      for (const [k, v] of Object.entries(data)) {
        const elem = document.getElementById(k);
        if (elem) {
          elem.outerHTML = v;
        }
      }
      if (triggerHighlight) {
        highlight();
      }
    },
    onError: (event) => {
      console.error("SSE error:", event);
    },
  });

  sseClient.connect();
};

const initCurrentTime = () => {
  const currentTimeElem = document.querySelector("#currentTime");
  if (!currentTimeElem) {
    console.error("Current time element could not be found!");
    return;
  }
  setInterval(() => {
    const currentTime = new Date();

    const dateOptions = {
      timeZone: "Asia/Jakarta",
      day: "2-digit",
      month: "long",
      year: "numeric",
    };
    const timeOptions = {
      timeZone: "Asia/Jakarta",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    };

    const formattedDate = currentTime.toLocaleDateString("en-GB", dateOptions);
    const formattedTime = currentTime.toLocaleTimeString("en-GB", timeOptions);

    currentTimeElem.textContent = `${formattedDate} ${formattedTime}`;
  }, 1000);
};

const tableToImage = (year, month) => {
  const prevHiglightedRows = document.querySelectorAll(
    "#poop-log > table > tbody > tr.highlighted"
  );
  prevHiglightedRows.forEach((e) => e.classList.remove("highlighted"));

  const node = document.querySelector("#poop-log");
  if (!node) {
    console.error("Table could not be found!");
    return;
  }
  domtoimage
    .toPng(node, {
      bgcolor: "white",
    })
    .then((dataUrl) => fetch(dataUrl))
    .then((res) => res.blob())
    .then((blob) => {
      const downloadButton = document.createElement("a");
      const filename = `poop-log-${year}${
        month ? "-" + month.padStart(2, "0") : ""
      }.png`;
      downloadButton.download = filename;
      downloadButton.href = URL.createObjectURL(blob);
      prevHiglightedRows.forEach((e) => e.classList.add("highlighted"));
      if (isIOS && sseClient) {
        sseClient.pause();
        setTimeout(() => {
          sseClient.resume();
        }, 1000);
      }
      downloadButton.click();
    });
  // .then(function (dataUrl) {
  //   const downloadButton = document.createElement("a");
  //   const filename = `poop-log-${year}${
  //     month ? "-" + month.padStart(2, "0") : ""
  //   }.png`;
  //   downloadButton.download = filename;
  //   downloadButton.href = dataUrl;
  //   prevHiglightedRows.forEach((e) => e.classList.add("highlighted"));
  //   if (isIOS && sseClient) {
  //     sseClient.pause();
  //     setTimeout(() => {
  //       sseClient.resume();
  //     }, 1000);
  //   }
  //   downloadButton.click();
  //   downloadButton.remove();
  // });
};
