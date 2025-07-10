const service = "io.github.muesli.DeckMaster";
const path = "/Monitor";
const method = "ActiveWindowChanged";

function activeWindowChanged(window) {
    const name = window.resourceName + "." + window.resourceClass;
    const id = window.id.toString();
    print("activated: " + name + ' ' + id);
    callDBus(service, path, service, method, name, id);
}

workspace.windowActivated.connect(activeWindowChanged);
print("activated: DeckMaster");
