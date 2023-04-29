

function initJS() {
    // Get userconfig
    const url = "/api/v1/settings";
    let request = new XMLHttpRequest();

    request.open("GET", url);
    request.setRequestHeader("Accept", "application/json")
    request.onload = function () {
        if (request.status === 200) {
            let settings = JSON.parse(request.responseText);
            console.log(settings)

            // DarkMode
            if (settings.darkMode) {
                document.querySelector("#darkmode_dark").checked = true;
            } else {
                document.querySelector("#darkmode_light").checked = true;
            }
            // Language
            document.querySelector("#lang_"+settings.language).checked = true;
        }
    }
    request.send(null);
}

document.addEventListener('DOMContentLoaded', initJS);
console.log("EventListener added");