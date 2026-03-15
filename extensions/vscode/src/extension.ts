import * as vscode from 'vscode';
import { SnippetTreeProvider } from './tree/snippetTreeProvider';
import { createWatcher } from './watcher';
import { disposeOutputChannel, resolveVaultPath } from './cli';
import { showSnippet } from './commands/showSnippet';
import { insertSnippet } from './commands/insertSnippet';
import { copySnippet } from './commands/copySnippet';
import { deleteSnippet } from './commands/deleteSnippet';
import { newSnippet, newSnippetFromSelection, addFile } from './commands/createSnippet';
import { editSnippet, editSnippetMeta } from './commands/editSnippet';
import { searchSnippets } from './commands/searchSnippet';
import { syncVault } from './commands/syncVault';

export async function activate(context: vscode.ExtensionContext): Promise<void> {
    let vaultPath: string;
    try {
        vaultPath = await resolveVaultPath();
    } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        const action = await vscode.window.showErrorMessage(
            `Trove CLI not found or failed: ${msg}`,
            'Open Settings',
        );
        if (action === 'Open Settings') {
            vscode.commands.executeCommand('workbench.action.openSettings', 'trove.cliPath');
        }
        return;
    }

    const treeProvider = new SnippetTreeProvider();
    const treeView = vscode.window.createTreeView('troveSnippets', {
        treeDataProvider: treeProvider,
        showCollapseAll: true,
    });
    context.subscriptions.push(treeView);

    const watcherDisposables = createWatcher(vaultPath, treeProvider);
    context.subscriptions.push(...watcherDisposables);

    const commands: [string, (...args: unknown[]) => unknown][] = [
        ['trove.newSnippet', () => newSnippet(vaultPath, treeProvider)],
        ['trove.newSnippetFromSelection', () => newSnippetFromSelection(vaultPath, treeProvider)],
        ['trove.addFile', () => addFile(vaultPath, treeProvider)],
        ['trove.editSnippet', (item?: unknown) => editSnippet(vaultPath, item as never)],
        ['trove.editSnippetMeta', (item?: unknown) => editSnippetMeta(treeProvider, item as never)],
        ['trove.deleteSnippet', (item?: unknown) => deleteSnippet(treeProvider, item as never)],
        ['trove.showSnippet', (item?: unknown) => showSnippet(vaultPath, item as never)],
        ['trove.insertSnippet', (item?: unknown) => insertSnippet(item as never)],
        ['trove.searchSnippets', () => searchSnippets(vaultPath, treeProvider)],
        ['trove.syncVault', () => syncVault()],
        ['trove.refreshTree', () => treeProvider.refresh()],
        ['trove.copySnippet', (item?: unknown) => copySnippet(item as never)],
    ];

    for (const [id, handler] of commands) {
        context.subscriptions.push(vscode.commands.registerCommand(id, handler));
    }
}

export function deactivate(): void {
    disposeOutputChannel();
}
