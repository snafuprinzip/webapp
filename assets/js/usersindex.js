const byName = 0;
const byID = 1;
const byEmail = 2;
const bySessions = 3;

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
        (s1, s2) => (s1.sessions < s2.sessions) ? -1 : (s1.sessions > s2.sessions) ? 1 : 0);
    printListe(sortedUsers);
}


function deleteUser(id) {
    const url = "/api/v1/users/" + id;
    let request = new XMLHttpRequest();

    request.open("DELETE", url);
    request.setRequestHeader("Accept", "application/json")
    request.onload = function () {
        if (request.status == 200) {
            let user = JSON.parse(request.responseText)
        }
    }
    request.send(null);
    if (request.status == 200) {

    }


    return ownerName

}

function printListe(users) {
    let usersTable = document.querySelectorAll("#userslist");
    usersTable.innerHTML = "";

    for (let s = 0; s < users.length; s++) {
        let user= users[s];
        console.log(user);
        let tr = document.createElement("tr");

        let tdid = document.createElement("td");
        tdid.innerHTML = user.id;
        tr.appendChild(tdid);

        let tdname = document.createElement("td");
        tdname.innerHTML = user.username;
        tr.appendChild(tdname);

        let tdemail = document.createElement("td");
        tdemail.innerHTML = user.email;
        tr.appendChild(tdemail);

        let tdsessions = document.createElement("td");
        let sessionsstr = "";
        for (let r = 0; r < user.sessions.length; r++) {
            sessionsstr += user.sessions[r].id + "<br>";
        }
        tdpsessions.innerHTML = sessionsstr;
        tr.appendChild(tdsessions);

        // Listitem Buttons (edit, delete)
        let tdbuttons = document.createElement("td");
        let ul = document.createElement("ul");
        ul.setAttribute("class", "list-inline m-0");
        let li = document.createElement("li");
        li.setAttribute("class", "list-inline-item");
        let a = document.createElement("a");
        a.setAttribute("href", "/users/" + user.id);
        a.setAttribute("class", "btn buttonaction btn-success btn-sm rounded-0");
        a.setAttribute("role", "button");
        a.setAttribute("data-toggle", "tooltip");
        a.setAttribute("data-placement", "top");
        a.setAttribute("title", "Edit");
        let i = document.createElement("i");
        i.setAttribute("class", "fa fa-edit");
        a.appendChild(i);
        li.appendChild(a);

        let a2 = document.createElement("a");
        // a2.setAttribute("href", "/userrm/" + user.ID);
        a2.setAttribute("onclick", "deleteUser("+user.id+")");
        a2.setAttribute("class", "btn buttonaction btn-danger btn-sm rounded-0");
        a2.setAttribute("role", "button");
        a2.setAttribute("data-toggle", "tooltip");
        a2.setAttribute("data-placement", "top");
        a2.setAttribute("title", "Delete");
        let i2 = document.createElement("i");
        i2.setAttribute("class", "fa fa-trash");
        a2.appendChild(i2);
        li.appendChild(a2);


        ul.appendChild(li);
        tdbuttons.appendChild(ul);
        tr.appendChild(tdbuttons);

        usersTable.appendChild(tr);
    }
}

function getData(sorting) {
    const url = "/api/v1/users/";
    let request = new XMLHttpRequest();

    request.open("GET", url);
    request.setRequestHeader("Accept", "application/json")
    request.onload = function () {
        if (request.status == 200) {
            if (sorting == byID) {
                sortByID(request.responseText);
            } else if (sorting == byEmail) {
                sortByEmail(request.responseText);
            } else if (sorting == bySessions) {
                sortBySessions(request.responseText);
            } else {
                sortByName(request.responseText);
            }
        }
    }
    request.send(null);
}

function initScript() {
    getData(byName);
}

window.onload = initScript();