import * as vscode from 'vscode';
import * as cli from '../cli';
import { TroveSnippet } from '../types';
import { SnippetTreeItem } from '../tree/snippetTreeItem';
import { SnippetTreeProvider } from '../tree/snippetTreeProvider';
import { getShowNotifications } from '../config';

export async function deleteSnippet(
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

    const answer = await vscode.window.showWarningMessage(
        `Delete snippet "${snippet.id}"?`,
        { modal: true },
        'Delete',
    );

    if (answer !== 'Delete') { return; }

    try {
        await cli.remove(snippet.id);
        treeProvider.refresh();
        if (getShowNotifications()) {
            vscode.window.showInformationMessage(`Deleted ${snippet.id}`);
        }
    } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage(`Trove: ${msg}`);
    }
}
