import * as vscode from 'vscode';
import { TroveSnippet } from '../types';
import { LanguageTreeItem, SnippetTreeItem } from './snippetTreeItem';
import * as cli from '../cli';

export class SnippetTreeProvider implements vscode.TreeDataProvider<vscode.TreeItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<void>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    private snippets: TroveSnippet[] = [];

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: vscode.TreeItem): vscode.TreeItem {
        return element;
    }

    async getChildren(element?: vscode.TreeItem): Promise<vscode.TreeItem[]> {
        if (!element) {
            try {
                this.snippets = await cli.list();
            } catch (err) {
                this.snippets = [];
                const msg = err instanceof Error ? err.message : String(err);
                vscode.window.showErrorMessage(`Trove: ${msg}`);
                return [];
            }

            const byLang = new Map<string, TroveSnippet[]>();
            for (const s of this.snippets) {
                const group = byLang.get(s.language) || [];
                group.push(s);
                byLang.set(s.language, group);
            }

            const langs = [...byLang.keys()].sort();
            return langs.map(lang => new LanguageTreeItem(lang, byLang.get(lang)!.length));
        }

        if (element instanceof LanguageTreeItem) {
            return this.snippets
                .filter(s => s.language === element.language)
                .sort((a, b) => a.name.localeCompare(b.name))
                .map(s => new SnippetTreeItem(s));
        }

        return [];
    }

    getAllSnippets(): TroveSnippet[] {
        return this.snippets;
    }
}
