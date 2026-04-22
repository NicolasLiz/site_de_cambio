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

var dataCache = []

async function getGraphData(symbol) {
    for (const data of dataCache) {
        if (data[0].Symbol === symbol) {
            return data
        }
    }

    const response = await fetch("/graph-data?symbol=" + symbol, {
        headers: {
            "content-type": "appliation/x-www-form-urlencoded"
        },
        method: "POST"
    })
    const data = await response.json()
    dataCache.push(data)
    return data
}


function updateGraph(which) {
    if (which == 1) {
        fromGraph.data.labels = []
        fromGraph.data.datasets.forEach((dataset) => {dataset.data = []})
        getGraphData(document.getElementById("from").value).then(function(res) {
            res.forEach((data) => {
                if (data.Value != 0) {
                    fromGraph.data.labels.push(data.Date)
                    fromGraph.data.datasets.forEach((dataset) => {
                        dataset.data.push(data.Value)
                    })
                }
            })
            fromGraph.update()
        })
    } else {
        toGraph.data.labels = []
        toGraph.data.datasets.forEach((dataset) => {dataset.data = []})
        getGraphData(document.getElementById("to").value).then(function(res) {
            res.forEach((data) => {
                if (data.Value != 0) {
                    toGraph.data.labels.push(data.Date)
                    toGraph.data.datasets.forEach((dataset) => {
                        dataset.data.push(data.Value)
                    })
                }
            })
            toGraph.update()
        })
    }
}
