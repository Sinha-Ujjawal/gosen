function mkHeader(header) {
    let item = document.createElement("span");
    let boldHeader = document.createElement("strong");
    boldHeader.appendChild(document.createTextNode(header));
    item.appendChild(boldHeader);
    item.appendChild(document.createElement("br"));
    return item;
}

async function search(prompt, topN) {
    if (topN === undefined || isNaN(topN) || topN <= 0) {
        topN = 1
    }
    const query = {
        "search": prompt,
        "topN": topN,
    }
    const results = document.getElementById("results")
    results.innerHTML = "";
    const response = await fetch("/api/search", {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(query),
    });
    const jsonArr = await response.json();
    results.innerHTML = "";
    let headerNode = mkHeader(`Showing top ${Math.min(topN, jsonArr.length)} results:`);
    results.appendChild(headerNode);
    for (let i = 0; i < jsonArr.length; i++) {
        let { docId, _score } = jsonArr[i];
        let item = document.createElement("span");
        item.appendChild(document.createTextNode(docId));
        item.appendChild(document.createElement("br"));
        results.appendChild(item);
    }
}

let query = document.getElementById("query");
let topN = document.getElementById("topN");
let searchButton = document.getElementById("search");
let currentSearch = Promise.resolve()

searchButton.onclick = function (e) {
    currentSearch.then(() => search(query.value, topN.value * 1));
}
