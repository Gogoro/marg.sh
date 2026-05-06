import AppKit
import SwiftUI

@main
struct MargMain {
    static func main() {
        let application = NSApplication.shared
        let delegate = MargAppDelegate()
        application.delegate = delegate
        application.setActivationPolicy(.regular)
        application.run()
    }
}

final class MargAppDelegate: NSObject, NSApplicationDelegate {
    let appState = AppState()
    var window: NSWindow?

    func applicationDidFinishLaunching(_ notification: Notification) {
        installMenuBar()

        let rootView = ContentView()
            .environmentObject(appState)

        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 1100, height: 720),
            styleMask: [.titled, .closable, .miniaturizable, .resizable, .fullSizeContentView],
            backing: .buffered,
            defer: false
        )
        window.center()
        window.setFrameAutosaveName("MargMainWindow")
        window.title = "marg"
        window.minSize = NSSize(width: 760, height: 520)
        window.contentView = NSHostingView(rootView: rootView)
        window.makeKeyAndOrderFront(nil)
        self.window = window

        NSApp.activate(ignoringOtherApps: true)
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        return true
    }

    private func installMenuBar() {
        let mainMenu = NSMenu()

        let appMenuItem = NSMenuItem()
        let appMenu = NSMenu()
        appMenu.addItem(NSMenuItem(
            title: "About marg",
            action: #selector(NSApplication.orderFrontStandardAboutPanel(_:)),
            keyEquivalent: ""
        ))
        appMenu.addItem(NSMenuItem.separator())
        appMenu.addItem(NSMenuItem(
            title: "Hide marg",
            action: #selector(NSApplication.hide(_:)),
            keyEquivalent: "h"
        ))
        appMenu.addItem(NSMenuItem(
            title: "Quit marg",
            action: #selector(NSApplication.terminate(_:)),
            keyEquivalent: "q"
        ))
        appMenuItem.submenu = appMenu
        mainMenu.addItem(appMenuItem)

        let fileMenuItem = NSMenuItem()
        let fileMenu = NSMenu(title: "File")
        fileMenuItem.submenu = fileMenu

        let findItem = NSMenuItem(
            title: "Find File…",
            action: #selector(handleFindFile),
            keyEquivalent: "p"
        )
        findItem.target = self
        fileMenu.addItem(findItem)

        let saveItem = NSMenuItem(
            title: "Save",
            action: #selector(handleSave),
            keyEquivalent: "s"
        )
        saveItem.target = self
        fileMenu.addItem(saveItem)

        fileMenu.addItem(NSMenuItem.separator())

        let reloadItem = NSMenuItem(
            title: "Reload Index",
            action: #selector(handleReloadIndex),
            keyEquivalent: "r"
        )
        reloadItem.keyEquivalentModifierMask = [.command, .shift]
        reloadItem.target = self
        fileMenu.addItem(reloadItem)

        let discardItem = NSMenuItem(
            title: "Discard External Changes",
            action: #selector(handleDiscard),
            keyEquivalent: "e"
        )
        discardItem.keyEquivalentModifierMask = [.command, .shift]
        discardItem.target = self
        fileMenu.addItem(discardItem)

        mainMenu.addItem(fileMenuItem)

        let editMenuItem = NSMenuItem()
        let editMenu = NSMenu(title: "Edit")
        editMenu.addItem(NSMenuItem(
            title: "Undo",
            action: Selector(("undo:")),
            keyEquivalent: "z"
        ))
        let redoItem = NSMenuItem(
            title: "Redo",
            action: Selector(("redo:")),
            keyEquivalent: "z"
        )
        redoItem.keyEquivalentModifierMask = [.command, .shift]
        editMenu.addItem(redoItem)
        editMenu.addItem(NSMenuItem.separator())
        editMenu.addItem(NSMenuItem(
            title: "Cut",
            action: #selector(NSText.cut(_:)),
            keyEquivalent: "x"
        ))
        editMenu.addItem(NSMenuItem(
            title: "Copy",
            action: #selector(NSText.copy(_:)),
            keyEquivalent: "c"
        ))
        editMenu.addItem(NSMenuItem(
            title: "Paste",
            action: #selector(NSText.paste(_:)),
            keyEquivalent: "v"
        ))
        editMenu.addItem(NSMenuItem(
            title: "Select All",
            action: #selector(NSText.selectAll(_:)),
            keyEquivalent: "a"
        ))
        editMenuItem.submenu = editMenu
        mainMenu.addItem(editMenuItem)

        NSApp.mainMenu = mainMenu
    }

    @objc private func handleFindFile() {
        appState.showingPicker = true
    }

    @objc private func handleSave() {
        appState.saveCurrentFile()
    }

    @objc private func handleReloadIndex() {
        appState.refreshIndex()
    }

    @objc private func handleDiscard() {
        appState.discardAndReload()
    }
}
