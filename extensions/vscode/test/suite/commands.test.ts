import * as assert from 'assert';

suite('Commands', () => {
    test('modules can be imported', () => {
        // Verify all command modules export the expected functions
        const createSnippet = require('../../src/commands/createSnippet');
        assert.ok(typeof createSnippet.newSnippet === 'function');
        assert.ok(typeof createSnippet.newSnippetFromSelection === 'function');
        assert.ok(typeof createSnippet.addFile === 'function');

        const editSnippet = require('../../src/commands/editSnippet');
        assert.ok(typeof editSnippet.editSnippet === 'function');
        assert.ok(typeof editSnippet.editSnippetMeta === 'function');

        const deleteSnippet = require('../../src/commands/deleteSnippet');
        assert.ok(typeof deleteSnippet.deleteSnippet === 'function');

        const showSnippet = require('../../src/commands/showSnippet');
        assert.ok(typeof showSnippet.showSnippet === 'function');

        const insertSnippet = require('../../src/commands/insertSnippet');
        assert.ok(typeof insertSnippet.insertSnippet === 'function');

        const copySnippet = require('../../src/commands/copySnippet');
        assert.ok(typeof copySnippet.copySnippet === 'function');

        const searchSnippet = require('../../src/commands/searchSnippet');
        assert.ok(typeof searchSnippet.searchSnippets === 'function');

        const syncVault = require('../../src/commands/syncVault');
        assert.ok(typeof syncVault.syncVault === 'function');
    });
});
