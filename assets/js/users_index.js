const markup = (user, sessions, editLabel, deleteLabel, confirmationText) => {
    return `
    <tr>
        <td>${user.id}</td>
        <td>${user.username}</td>
        <td>${user.email}</td>
        <td>${sessions}</td>
        <td>
            <ul class="list-inline m-0">
                <li>
                    <a href="/users/${user.id}" class="btn buttonaction btn-success btn-sm rounded-0"
                        role="button" data-toggle="tooltip" data-placement="top" title="${editLabel}">
                        <i class="fa fa-edit"></i>
                    </a>
<!--                    <a href="/userrm/${user.id}" onclick="deleteUser(${user.id})" class="btn buttonaction btn-danger btn-sm rounded-0"-->
                    <a onclick="deleteUser('${user.id}', '${confirmationText}')" class="btn buttonaction btn-danger btn-sm rounded-0"
                        id="delete${user.id}" role="button" data-toggle="tooltip" data-placement="top" title="${deleteLabel}">
                        <i class="fa fa-trash"></i>
                    </a>
                </li>            
            </ul>
        </td>
    </tr>    
`;
};

var language = "en";

function sortByName(response) {
    let users = JSON.parse(response);
    let sortedUsers = users.sort(
        (s1, s2) => (s1.username < s2.username) ? -1 : (s1.username > s2.username) ? 1 : 0);
    printListe(sortedUsers);
}
function sortByID(response) {
    let users = JSON.parse(response);
    let sortedUsers = users.sort(
        (s1, s2) => (s1.id < s2.id) ? -1 : (s1.id > s2.id) ? 1 : 0);
    printListe(sortedUsers);
}
function sortByEmail(response) {
    let users = JSON.parse(response);
    let sortedUsers = users.sort(
        (s1, s2) => (s1.email < s2.email) ? -1 : (s1.email > s2.email) ? 1 : 0);
    printListe(sortedUsers);
}
function sortBySessions(response) {
    let users = JSON.parse(response);
    let sortedUsers = users.sort(
        (s1, s2) => (s1.sessions[0] < s2.sessions[0]) ? -1 : (s1.sessions[0] > s2.sessions[0]) ? 1 : 0);
    printListe(sortedUsers);
}

function getUserConfig(userid) {
    const url = "/api/v1/settings/" + userid;
    let request = new XMLHttpRequest();

    request.open("GET", url);
    request.setRequestHeader("Accept", "application/json")
    request.onload = function () {
        if (request.status === 200) {
            return JSON.parse(request.responseText)
        }
    }
    request.send(null);
}

function deleteUser(id, confirmationMessage) {
    let confirmation = confirm(confirmationMessage)
    if (confirmation) {
        const url = "/api/v1/users/" + id;
        let request = new XMLHttpRequest();

        request.open("DELETE", url);
        request.setRequestHeader("Accept", "application/json")
        request.onload = function () {
            if (request.status === 204) {
                window.location.reload();
            }
        }
        request.send(null);
    }
}

function printListe(users) {
    let usersTable = document.querySelector("#userslist");
    usersTable.innerHTML = "";

    users.forEach(function (user) {
        let sessionsstr = "";
        for (let r = 0; r < user.sessions.length; r++) {
            sessionsstr += user.sessions[r].id + "<br>";
        }
        switch (language) {
            case "de":
                usersTable.innerHTML += markup(user, sessionsstr, "Bearbeiten", "Entfernen",
                    "Sind sie sicher, dass sie den Benutzer "+ user.username +" löschen möchten?");
                break;
            default:
                usersTable.innerHTML += markup(user, sessionsstr, "Edit", "Delete",
                    "Are you sure you want to delete the user "+user.username+"?");
        }
    });
    document.querySelector("#deleteadmin").hidden = true;
}

function getData(sorting) {
    const url = "/api/v1/users/";
    let request = new XMLHttpRequest();

    console.log(sorting);

    request.open("GET", url);
    request.setRequestHeader("Accept", "application/json")
    request.onload = function () {
        if (request.status === 200) {
            if (sorting === "id") {
                sortByID(request.responseText);
            } else if (sorting === "email") {
                sortByEmail(request.responseText);
            } else if (sorting === "sessions") {
                sortBySessions(request.responseText);
            } else {
                sortByName(request.responseText);
            }
        }
    }
    request.send(null);
}

function setLanguage(lang) {
    language = lang;
    getData("name");
}

function initJS() {
    getData();
}

document.addEventListener('DOMContentLoaded', initJS);
