import { enableProdMode, NgModuleRef } from '@angular/core';
import { platformBrowserDynamic } from '@angular/platform-browser-dynamic';
import { AppModule } from './app/app.module';

export function main() {
  enableProdMode();
  platformBrowserDynamic().bootstrapModule(AppModule)
    .catch(err => console.error(err));
}

if (document.readyState === 'complete') {
  main();
} else {
  document.addEventListener('DOMContentLoaded', main);
}
