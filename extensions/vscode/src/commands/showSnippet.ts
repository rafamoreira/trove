import * as vscode from 'vscode';
import { TroveSnippet } from '../types';
import { SnippetTreeItem } from '../tree/snippetTreeItem';

export async function showSnippet(vaultPath: string, snippetOrItem?: TroveSnippet | SnippetTreeItem): Promise<void> {
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
