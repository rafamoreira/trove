import * as vscode from 'vscode';
import * as cli from '../cli';
import { TroveSnippet } from '../types';
import { SnippetTreeItem } from '../tree/snippetTreeItem';

export async function insertSnippet(snippetOrItem?: TroveSnippet | SnippetTreeItem): Promise<void> {
    const snippet = snippetOrItem instanceof SnippetTreeItem
        ? snippetOrItem.snippet
        : snippetOrItem;

    if (!snippet) {
        vscode.window.showErrorMessage('No snippet selected');
        return;
    }

    const editor = vscode.window.activeTextEditor;
    if (!editor) {
        vscode.window.showErrorMessage('No active editor');
        return;
    }

    try {
        const result = await cli.show(snippet.id);
        await editor.edit(editBuilder => {
            editBuilder.insert(editor.selection.active, result.body);
        });
    } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage(`Trove: ${msg}`);
    }
}
