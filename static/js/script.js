var fromGraph = new Chart("fromGraph", {
    type: "line",
    data: {
        labels: [],
        datasets: [{
            backgroundColor:"rgba(0,0,255,1.0)",
            borderColor: "rgba(0,0,255,0.1)",
            data: []
        }]
    },
    options: {}
})

var toGraph = new Chart("toGraph", {
    type: "line",
    data: {
        labels: [],
        datasets: [{
            backgroundColor:"rgba(0,0,255,1.0)",
            borderColor: "rgba(0,0,255,0.1)",
            data: []
        }]
    },
    options: {}
})

async function getGraphData(symbol) {
    const response = await fetch("/graph-data?symbol=" + symbol, {
        headers: {
            "content-type": "appliation/x-www-form-urlencoded"
        },
        method: "POST"
    })
    const data = await response.json()
    return data
}


function updateGraph(which) {
    if (which == 1) {
        fromGraph.data.labels = []
        fromGraph.data.datasets.forEach((dataset) => {dataset.data = []})
        getGraphData(document.getElementById("from").value).then(function(res) {
            for (var i = 0; i < 12; i++) {
                if (res[i].Value != 0) {
                    fromGraph.data.labels.push(res[i].Date)
                    fromGraph.data.datasets.forEach((dataset) => {
                        dataset.data.push(res[i].Value)
                    })
                }
            }
            fromGraph.update()
        })
    } else {
        toGraph.data.labels = []
        toGraph.data.datasets.forEach((dataset) => {dataset.data = []})
        getGraphData(document.getElementById("to").value).then(function(res) {
            for (var i = 0; i < 12; i++) {
                if (res[i].Value != 0) {
                    toGraph.data.labels.push(res[i].Date)
                    toGraph.data.datasets.forEach((dataset) => {
                        dataset.data.push(res[i].Value)
                    })
                }
            }
            toGraph.update()
        })
    }
}
