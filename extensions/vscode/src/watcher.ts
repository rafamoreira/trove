import * as vscode from 'vscode';
import { SnippetTreeProvider } from './tree/snippetTreeProvider';
import * as cli from './cli';
import { getSyncOnSave } from './config';

export function createWatcher(
    vaultPath: string,
    treeProvider: SnippetTreeProvider,
): vscode.Disposable[] {
    const pattern = new vscode.RelativePattern(vaultPath, '**/*');
    const watcher = vscode.workspace.createFileSystemWatcher(pattern);

    let debounceTimer: ReturnType<typeof setTimeout> | undefined;

    function scheduleRefresh() {
        if (debounceTimer) { clearTimeout(debounceTimer); }
        debounceTimer = setTimeout(() => treeProvider.refresh(), 300);
    }

    watcher.onDidCreate(scheduleRefresh);
    watcher.onDidChange(scheduleRefresh);
    watcher.onDidDelete(scheduleRefresh);

    const disposables: vscode.Disposable[] = [watcher];

    const saveListener = vscode.workspace.onDidSaveTextDocument(async (doc) => {
        if (!getSyncOnSave()) { return; }
        if (doc.uri.fsPath.startsWith(vaultPath) && !doc.uri.fsPath.includes('.git')) {
            try {
                await cli.sync();
            } catch {
                // sync errors are non-critical when auto-syncing
            }
        }
    });
    disposables.push(saveListener);

    return disposables;
}
