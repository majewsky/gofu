(() => {
  const $ = selector => document.querySelector(selector);

  const handleResponse = (promise, action) => {
    const error = promise.then(response => {
      if (!response.ok) {
        throw new Error(`HTTP error, status = ${response.status} from ${response.url}`);
      }
      const title = response.headers.get("X-Mdedit-Path");
      if (title !== null && title !== "") {
        window.document.title = "mdedit: " + title;
        $("span#title").innerText = title;
      }
      return response.text();
    }).then(body => {
      action(body);
      return null;
    }).catch((error) => {
      return error;
    }).then((error) => {
      if (error != null) {
        console.log(error);
        $("span#status").innerText = `Error: ${error.message}`;
      }
    });
  };

  // handler for receiving a rendered HTML from the server
  const receiveHTML = body => {
    $("div#preview").innerHTML = body;
    $("span#status").innerText = "Saved";
  };

  // whenever the <textedit> changes, wait 3 seconds and then save all changes
  let timeoutID = null;
  const onTextChange = event => {
    if (timeoutID !== null) {
      window.clearTimeout(timeoutID);
    }
    timeoutID = window.setTimeout(uploadMarkdown, 1000);
    $("span#status").innerText = "Changed";
  };
  const uploadMarkdown = () => {
    window.clearTimeout(timeoutID);
    timeoutID = null;
    $("span#status").innerText = "Saving...";

    const opts = {
      method: "PUT",
      body: $("textarea#editor").value,
    };
    handleResponse(fetch("/data.md", opts), receiveHTML);
  };

  // load Markdown and rendered HTML on startup
  handleResponse(fetch("/data.md"), body => {
    const editor = $("textarea#editor");
    $("textarea#editor").value = body;
    handleResponse(fetch("/data.html"), receiveHTML);

    // avoid a useless upload on startup by attaching this only after the initial load is done
    for (const eventType of ["change", "input", "textInput"]) {
      editor.addEventListener(eventType, onTextChange);
    }
  });

})();
