import { Component, Input, Output, EventEmitter } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Observable } from 'rxjs';
import { Model, Provider } from '../api.service';

@Component({
  selector: 'app-model-form',
  templateUrl: './model-form.component.html',
  styleUrls: ['./model-form.component.scss']
})
export class ModelFormComponent {
  @Input() model: Model | null = null;
  @Input() providers: Provider[] = [];
  @Output() saved = new EventEmitter<Model>();
  @Output() cancelled = new EventEmitter<void>();

  modelForm: FormGroup;
  isEditing = false;
  loading = false;
  error: string | null = null;

  constructor(private fb: FormBuilder) {
    this.initializeForm();
  }

  ngOnChanges(): void {
    if (this.model) {
      this.isEditing = true;
      this.populateForm();
    } else {
      this.isEditing = false;
      this.resetForm();
    }
  }

  private initializeForm(): void {
    this.modelForm = this.fb.group({
      provider_id: ['', [Validators.required, Validators.min(1)]],
      model_id: ['', [Validators.required, Validators.minLength(1), Validators.maxLength(100)]],
      name: ['', [Validators.required, Validators.minLength(2), Validators.maxLength(100)]],
      description: ['', Validators.maxLength(500)],
      version: ['', Validators.maxLength(50)],
      architecture: ['', Validators.maxLength(100)],
      parameter_count: [null, Validators.min(0)],
      context_window_tokens: [null, Validators.min(0)],
      max_output_tokens: [null, Validators.min(0)],
      is_multimodal: [false],
      supports_vision: [false],
      supports_audio: [false],
      supports_video: [false],
      supports_reasoning: [false],
      open_source: [false],
      deprecated: [false],
      tags: [[]],
      language_support: [[]],
      use_case: ['', Validators.maxLength(200)],
      verification_status: ['pending', Validators.required],
      overall_score: [0, Validators.min(0), Validators.max(100)],
      code_capability_score: [0, Validators.min(0), Validators.max(100)],
      responsiveness_score: [0, Validators.min(0), Validators.max(100)],
      reliability_score: [0, Validators.min(0), Validators.max(100)],
      feature_richness_score: [0, Validators.min(0), Validators.max(100)],
      value_proposition_score: [0, Validators.min(0), Validators.max(100)]
    });
  }

  private populateForm(): void {
    if (this.model) {
      this.modelForm.patchValue({
        provider_id: this.model.provider_id,
        model_id: this.model.model_id,
        name: this.model.name,
        description: this.model.description || '',
        version: this.model.version || '',
        architecture: this.model.architecture || '',
        parameter_count: this.model.parameter_count || null,
        context_window_tokens: this.model.context_window_tokens || null,
        max_output_tokens: this.model.max_output_tokens || null,
        is_multimodal: this.model.is_multimodal || false,
        supports_vision: this.model.supports_vision || false,
        supports_audio: this.model.supports_audio || false,
        supports_video: this.model.supports_video || false,
        supports_reasoning: this.model.supports_reasoning || false,
        open_source: this.model.open_source || false,
        deprecated: this.model.deprecated || false,
        tags: this.model.tags || [],
        language_support: this.model.language_support || [],
        use_case: this.model.use_case || '',
        verification_status: this.model.verification_status || 'pending',
        overall_score: this.model.overall_score || 0,
        code_capability_score: this.model.code_capability_score || 0,
        responsiveness_score: this.model.responsiveness_score || 0,
        reliability_score: this.model.reliability_score || 0,
        feature_richness_score: this.model.feature_richness_score || 0,
        value_proposition_score: this.model.value_proposition_score || 0
      });
    }
  }

  private resetForm(): void {
    this.modelForm.reset({
      provider_id: '',
      model_id: '',
      name: '',
      description: '',
      version: '',
      architecture: '',
      parameter_count: null,
      context_window_tokens: null,
      max_output_tokens: null,
      is_multimodal: false,
      supports_vision: false,
      supports_audio: false,
      supports_video: false,
      supports_reasoning: false,
      open_source: false,
      deprecated: false,
      tags: [],
      language_support: [],
      use_case: '',
      verification_status: 'pending',
      overall_score: 0,
      code_capability_score: 0,
      responsiveness_score: 0,
      reliability_score: 0,
      feature_richness_score: 0,
      value_proposition_score: 0
    });
  }

  onSubmit(): void {
    if (this.modelForm.invalid) {
      this.markFormGroupAsTouched(this.modelForm);
      return;
    }

    this.loading = true;
    this.error = null;

    const formValue = this.modelForm.value;
    const modelData: Partial<Model> = {
      ...formValue,
      parameter_count: formValue.parameter_count || undefined,
      context_window_tokens: formValue.context_window_tokens || undefined,
      max_output_tokens: formValue.max_output_tokens || undefined
    };

    this.saved.emit(modelData as Model);
  }

  onCancel(): void {
    this.cancelled.emit();
  }

  getErrorMessage(controlName: string): string {
    const control = this.modelForm.get(controlName);
    if (!control || !control.errors || !control.touched) {
      return '';
    }

    const errors = control.errors;
    if (errors['required']) {
      return 'This field is required';
    }
    if (errors['minlength']) {
      return `Minimum length is ${errors['minlength'].requiredLength}`;
    }
    if (errors['maxlength']) {
      return `Maximum length is ${errors['maxlength'].requiredLength}`;
    }
    if (errors['min']) {
      return `Minimum value is ${errors['min'].min}`;
    }
    if (errors['max']) {
      return `Maximum value is ${errors['max'].max}`;
    }

    return 'Invalid value';
  }

  private markFormGroupAsTouched(formGroup: FormGroup): void {
    Object.keys(formGroup.controls).forEach(key => {
      const control = formGroup.get(key);
      control?.markAsTouched();
    });
  }

  addTag(event: any): void {
    const input = event.input;
    const value = event.value;

    if ((value || '').trim()) {
      const tags = this.modelForm.get('tags')?.value || [];
      tags.push(value.trim());
      this.modelForm.get('tags')?.setValue(tags);
    }

    if (input) {
      input.value = '';
    }
  }

  removeTag(tag: string): void {
    const tags = this.modelForm.get('tags')?.value || [];
    const index = tags.indexOf(tag);
    if (index >= 0) {
      tags.splice(index, 1);
      this.modelForm.get('tags')?.setValue(tags);
    }
  }

  addLanguage(event: any): void {
    const input = event.input;
    const value = event.value;

    if ((value || '').trim()) {
      const languages = this.modelForm.get('language_support')?.value || [];
      languages.push(value.trim());
      this.modelForm.get('language_support')?.setValue(languages);
    }

    if (input) {
      input.value = '';
    }
  }

  removeLanguage(language: string): void {
    const languages = this.modelForm.get('language_support')?.value || [];
    const index = languages.indexOf(language);
    if (index >= 0) {
      languages.splice(index, 1);
      this.modelForm.get('language_support')?.setValue(languages);
    }
  }
}