/** Creates header node given a header text
 * 
 * @param {string} header - header text
 * @returns HTMLSpanElement - header as a span element
 */
function mkHeader(header) {
    const item = document.createElement("span");
    const boldHeader = document.createElement("strong");
    boldHeader.appendChild(document.createTextNode(header));
    item.appendChild(boldHeader);
    item.appendChild(document.createElement("br"));
    return item;
}

/** searches for a given prompt to /api/search server, and correspondingly updates the ui with the results
 * 
 * @param {string} prompt - query string
 * @param {integer} topN - top n results to show
 */
async function search(prompt, topN) {
    if (topN === undefined || isNaN(topN) || topN <= 0) {
        topN = 1
    }
    const query = {
        "search": prompt,
        "topN": topN,
    }
    const results = document.getElementById("results")
    if (results == null) {
        return
    }
    results.innerHTML = "";
    const response = await fetch("/api/search", {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(query),
    });
    /**
     * @type {[{docId: string, score: number}]}
     */
    const jsonArr = await response.json();
    results.innerHTML = "";
    const headerNode = mkHeader(`Showing top ${Math.min(topN, jsonArr.length)} results:`);
    results.appendChild(headerNode);
    for (let i = 0; i < jsonArr.length; i++) {
        const { docId, score } = jsonArr[i];
        const item = document.createElement("span");
        item.appendChild(document.createTextNode(docId));
        item.appendChild(document.createElement("br"));
        results.appendChild(item);
    }
}

/** Main setup
 * 
 * @returns 
 */
function setup() {
    const query = document.getElementById("query");
    if (query === null) {
        console.log("query element not found!");
        return;
    }
    const topN = document.getElementById("topN");
    if (topN === null) {
        console.log("topN element not found!");
        return;
    }
    const searchButton = document.getElementById("search");
    if (searchButton === null) {
        console.log("searchButton element not found!");
        return;
    }
    const currentSearch = Promise.resolve();
    searchButton.onclick = function (e) {
        currentSearch.then(() => search(query.value, topN.value * 1));
    }
}

setup();
