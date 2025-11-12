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

const listenToPoopEvent = (period, year, month) => {
  const param = new URLSearchParams();
  param.append("period", period);
  param.append("year", year);
  if (month) {
    param.append("month", month);
  }
  const eventSource = new EventSource(`/sse?${param.toString()}`);
  eventSource.addEventListener("poopupdate", (event) => {
    const data = JSON.parse(event.data);
    for (const [k, v] of Object.entries(data)) {
      const elem = document.getElementById(k);
      elem.outerHTML = v;
    }
    highlight();
  });
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
    .then(function (dataUrl) {
      const downloadButton = document.querySelector("#download-button");
      downloadButton.download = `poop-log-${year}${
        !month ? "" : "-" + month.padStart(2, "0")
      }.png`;
      downloadButton.href = dataUrl;
      prevHiglightedRows.forEach((e) => e.classList.add("highlighted"));
    });
};
