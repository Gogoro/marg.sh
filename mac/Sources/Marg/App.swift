import AppKit
import SwiftUI

@main
struct MargApp: App {
    @StateObject private var appState = AppState()

    init() {
        NSApp?.setActivationPolicy(.regular)
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(appState)
                .frame(minWidth: 760, minHeight: 520)
                .onAppear {
                    NSApp.activate(ignoringOtherApps: true)
                }
        }
        .commands {
            CommandGroup(replacing: .newItem) { }

            CommandMenu("File") {
                Button("Find File…") {
                    appState.showingPicker = true
                }
                .keyboardShortcut("p", modifiers: .command)

                Button("Save") {
                    appState.saveCurrentFile()
                }
                .keyboardShortcut("s", modifiers: .command)
                .disabled(appState.currentFileURL == nil)

                Button("Reload Index") {
                    appState.refreshIndex()
                }
                .keyboardShortcut("r", modifiers: [.command, .shift])

                Divider()

                Button("Discard External Changes") {
                    appState.discardAndReload()
                }
                .keyboardShortcut("e", modifiers: [.command, .shift])
                .disabled(appState.currentFileURL == nil)
            }
        }
    }
}
