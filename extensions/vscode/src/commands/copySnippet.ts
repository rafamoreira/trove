import * as vscode from 'vscode';
import * as cli from '../cli';
import { TroveSnippet } from '../types';
import { SnippetTreeItem } from '../tree/snippetTreeItem';
import { getShowNotifications } from '../config';

export async function copySnippet(snippetOrItem?: TroveSnippet | SnippetTreeItem): Promise<void> {
    const snippet = snippetOrItem instanceof SnippetTreeItem
        ? snippetOrItem.snippet
        : snippetOrItem;

    if (!snippet) {
        vscode.window.showErrorMessage('No snippet selected');
        return;
    }

    try {
        const result = await cli.show(snippet.id);
        await vscode.env.clipboard.writeText(result.body);
        if (getShowNotifications()) {
            vscode.window.showInformationMessage(`Copied ${snippet.id} to clipboard`);
        }
    } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage(`Trove: ${msg}`);
    }
}
