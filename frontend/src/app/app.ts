import { Component } from '@angular/core';
import { Layout } from './shared/layout/layout';

@Component({
  selector: 'app-root',
  imports: [Layout],
  template: `<app-layout />`,
})
export class App {}