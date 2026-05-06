import Foundation

final class FileWatcher {
    private let url: URL
    private let onChange: () -> Void
    private var source: DispatchSourceFileSystemObject?
    private var fileDescriptor: Int32 = -1

    init(url: URL, onChange: @escaping () -> Void) {
        self.url = url
        self.onChange = onChange
    }

    func start() {
        let fd = open(url.path, O_EVTONLY)
        guard fd >= 0 else { return }
        fileDescriptor = fd

        let source = DispatchSource.makeFileSystemObjectSource(
            fileDescriptor: fd,
            eventMask: [.write, .delete, .rename, .extend],
            queue: DispatchQueue.global(qos: .utility)
        )
        let onChange = self.onChange
        source.setEventHandler { [weak self] in
            onChange()
            // For atomic-rename saves the inode is gone; reopen against the path.
            self?.reopenIfNeeded()
        }
        source.setCancelHandler { [weak self] in
            if let fd = self?.fileDescriptor, fd >= 0 {
                close(fd)
                self?.fileDescriptor = -1
            }
        }
        source.resume()
        self.source = source
    }

    func stop() {
        source?.cancel()
        source = nil
    }

    private func reopenIfNeeded() {
        guard let oldSource = source else { return }
        let oldFlags = oldSource.data
        if oldFlags.contains(.delete) || oldFlags.contains(.rename) {
            stop()
            start()
        }
    }

    deinit {
        stop()
    }
}
