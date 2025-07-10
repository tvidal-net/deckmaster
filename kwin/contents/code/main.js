const service = "io.github.muesli.DeckMaster";
const path = "/Monitor";
const method = "ActiveWindowChanged";

function activeWindowChanged(window) {
    const name = window.resourceName + "." + window.resourceClass;
    const id = window.id.toString();
    print("windowActivated: " + name + ' ' + id);
    callDBus(service, path, service, method, name, id);
}

const signal = workspace.windowActivated ?? workspace.clientActivated
signal.connect(activeWindowChanged);

print("enabled: DeckMaster");
