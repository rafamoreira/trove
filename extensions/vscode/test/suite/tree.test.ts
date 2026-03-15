import * as assert from 'assert';
import { LanguageTreeItem, SnippetTreeItem } from '../../src/tree/snippetTreeItem';
import { TroveSnippet } from '../../src/types';

suite('Tree Items', () => {
    test('LanguageTreeItem has correct label and count', () => {
        const item = new LanguageTreeItem('go', 5);
        assert.strictEqual(item.language, 'go');
        assert.strictEqual(item.label, 'go');
        assert.strictEqual(item.description, '5');
        assert.strictEqual(item.contextValue, 'language');
    });

    test('SnippetTreeItem uses snippet fields', () => {
        const snippet: TroveSnippet = {
            id: 'go/hello',
            name: 'hello',
            language: 'go',
            path: 'go/hello.go',
            meta_path: 'go/hello.toml',
            description: 'A greeting snippet',
            tags: ['util', 'greeting'],
            created: '2025-01-01T00:00:00Z',
        };

        const item = new SnippetTreeItem(snippet);
        assert.strictEqual(item.label, 'hello');
        assert.strictEqual(item.description, 'A greeting snippet');
        assert.strictEqual(item.contextValue, 'snippet');
        assert.strictEqual(item.snippet, snippet);
        assert.ok(item.tooltip?.toString().includes('go/hello'));
        assert.ok(item.tooltip?.toString().includes('Tags: util, greeting'));
    });
});
