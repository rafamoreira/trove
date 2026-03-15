import * as vscode from 'vscode';
import * as cli from '../cli';
import { TroveSnippet } from '../types';
import { SnippetTreeProvider } from '../tree/snippetTreeProvider';

interface SnippetQuickPickItem extends vscode.QuickPickItem {
    snippet: TroveSnippet;
}

export async function searchSnippets(
    vaultPath: string,
    treeProvider: SnippetTreeProvider,
): Promise<void> {
    const quickPick = vscode.window.createQuickPick<SnippetQuickPickItem>();
    quickPick.placeholder = 'Search snippets...';
    quickPick.matchOnDescription = true;
    quickPick.matchOnDetail = true;

    // Pre-load all snippets
    let allSnippets: TroveSnippet[] = treeProvider.getAllSnippets();
    if (!allSnippets.length) {
        try {
            allSnippets = await cli.list();
        } catch {
            allSnippets = [];
        }
    }

    function snippetToItem(s: TroveSnippet): SnippetQuickPickItem {
        return {
            label: `$(file-code) ${s.id}`,
            description: s.description,
            detail: s.tags?.length ? `Tags: ${s.tags.join(', ')}` : undefined,
            snippet: s,
        };
    }

    quickPick.items = allSnippets.map(snippetToItem);

    let searchTimeout: ReturnType<typeof setTimeout> | undefined;

    quickPick.onDidChangeValue((value) => {
        if (searchTimeout) { clearTimeout(searchTimeout); }
        if (!value.trim()) {
            quickPick.items = allSnippets.map(snippetToItem);
            return;
        }

        // If client-side filter yields results, use them; otherwise fall back to CLI search
        searchTimeout = setTimeout(async () => {
            const clientFiltered = allSnippets.filter(s =>
                s.id.includes(value) ||
                s.name.includes(value) ||
                s.description?.includes(value) ||
                s.tags?.some(t => t.includes(value))
            );

            if (clientFiltered.length >= 3) {
                quickPick.items = clientFiltered.map(snippetToItem);
                return;
            }

            try {
                const results = await cli.search(value);
                const items: SnippetQuickPickItem[] = results.map(r => ({
                    label: `$(file-code) ${r.snippet.id}`,
                    description: r.snippet.description,
                    detail: r.matches.map(m =>
                        m.line > 0 ? `L${m.line}: ${m.context}` : m.context
                    ).join(' | '),
                    snippet: r.snippet,
                }));
                if (items.length) {
                    quickPick.items = items;
                } else {
                    quickPick.items = clientFiltered.map(snippetToItem);
                }
            } catch {
                quickPick.items = clientFiltered.map(snippetToItem);
            }
        }, 200);
    });

    quickPick.onDidAccept(async () => {
        const selected = quickPick.selectedItems[0];
        quickPick.dispose();
        if (!selected) { return; }

        const action = await vscode.window.showQuickPick(
            ['Insert at cursor', 'Open in editor'],
            { placeHolder: 'What would you like to do?' },
        );

        if (action === 'Insert at cursor') {
            const editor = vscode.window.activeTextEditor;
            if (!editor) {
                vscode.window.showErrorMessage('No active editor');
                return;
            }
            try {
                const result = await cli.show(selected.snippet.id);
                await editor.edit(b => b.insert(editor.selection.active, result.body));
            } catch (err) {
                const msg = err instanceof Error ? err.message : String(err);
                vscode.window.showErrorMessage(`Trove: ${msg}`);
            }
        } else if (action === 'Open in editor') {
            const filePath = vscode.Uri.file(`${vaultPath}/${selected.snippet.path}`);
            const doc = await vscode.workspace.openTextDocument(filePath);
            await vscode.window.showTextDocument(doc);
        }
    });

    quickPick.onDidHide(() => quickPick.dispose());
    quickPick.show();
}
