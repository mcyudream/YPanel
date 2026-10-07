import { createRequire } from 'node:module';
const PACKAGE_NAME = createRequire(import.meta.url)('./package.json').name;
const AUTO_IMPORT_NAMES = [
    'usePagination',
];
export const ComposablesAutoImports = {
    [PACKAGE_NAME]: [...AUTO_IMPORT_NAMES],
};
