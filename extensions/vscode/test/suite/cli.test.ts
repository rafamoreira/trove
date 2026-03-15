import * as assert from 'assert';
import { TroveCliError } from '../../src/cli';

suite('CLI', () => {
    test('TroveCliError has correct properties', () => {
        const err = new TroveCliError('something failed', 1, 'stderr output');
        assert.strictEqual(err.message, 'something failed');
        assert.strictEqual(err.exitCode, 1);
        assert.strictEqual(err.stderr, 'stderr output');
        assert.strictEqual(err.name, 'TroveCliError');
        assert.ok(err instanceof Error);
    });
});
