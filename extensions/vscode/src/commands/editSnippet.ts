import * as vscode from 'vscode';
import * as cli from '../cli';
import { TroveSnippet } from '../types';
import { SnippetTreeItem } from '../tree/snippetTreeItem';
import { SnippetTreeProvider } from '../tree/snippetTreeProvider';
import { getShowNotifications } from '../config';

export async function editSnippet(
    vaultPath: string,
    snippetOrItem?: TroveSnippet | SnippetTreeItem,
): Promise<void> {
    const snippet = snippetOrItem instanceof SnippetTreeItem
        ? snippetOrItem.snippet
        : snippetOrItem;

    if (!snippet) {
        vscode.window.showErrorMessage('No snippet selected');
        return;
    }

    const filePath = vscode.Uri.file(`${vaultPath}/${snippet.path}`);
    const doc = await vscode.workspace.openTextDocument(filePath);
    await vscode.window.showTextDocument(doc);
}

export async function editSnippetMeta(
    treeProvider: SnippetTreeProvider,
    snippetOrItem?: TroveSnippet | SnippetTreeItem,
): Promise<void> {
    const snippet = snippetOrItem instanceof SnippetTreeItem
        ? snippetOrItem.snippet
        : snippetOrItem;

    if (!snippet) {
        vscode.window.showErrorMessage('No snippet selected');
        return;
    }

    const desc = await vscode.window.showInputBox({
        prompt: 'Description',
        value: snippet.description,
    });
    if (desc === undefined) { return; }

    const tags = await vscode.window.showInputBox({
        prompt: 'Tags (comma-separated)',
        value: snippet.tags?.join(', ') ?? '',
    });
    if (tags === undefined) { return; }

    try {
        await cli.editMeta(snippet.id, desc, tags);
        treeProvider.refresh();
        if (getShowNotifications()) {
            vscode.window.showInformationMessage(`Updated metadata for ${snippet.id}`);
        }
    } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage(`Trove: ${msg}`);
    }
}
