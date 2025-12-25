import { Component, Input, Output, EventEmitter } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Provider } from '../api.service';

@Component({
  selector: 'app-provider-form',
  templateUrl: './provider-form.component.html',
  styleUrls: ['./provider-form.component.scss']
})
export class ProviderFormComponent {
  @Input() provider: Provider | null = null;
  @Output() saved = new EventEmitter<Provider>();
  @Output() cancelled = new EventEmitter<void>();

  providerForm!: FormGroup;
  isEditing = false;
  loading = false;
  error: string | null = null;

  constructor(private fb: FormBuilder) {
    this.initializeForm();
  }

  ngOnChanges(): void {
    if (this.provider) {
      this.isEditing = true;
      this.populateForm();
    } else {
      this.isEditing = false;
      this.resetForm();
    }
  }

  private initializeForm(): void {
    this.providerForm = this.fb.group({
      name: ['', [Validators.required, Validators.minLength(2), Validators.maxLength(100)]],
      endpoint: ['', [Validators.required, Validators.pattern(/^https?:\/\/.+/)]],
      api_key_encrypted: ['', [Validators.required, Validators.minLength(10)]],
      description: ['', Validators.maxLength(500)],
      website: ['', Validators.pattern('^https?:\\/\\/.+')],
      support_email: ['', [Validators.email]],
      documentation_url: ['', Validators.pattern('^https?:\\/\\/.+')],
      is_active: [true],
      reliability_score: [0, Validators.min(0), Validators.max(100)],
      average_response_time_ms: [0, Validators.min(0)]
    });
  }

  private populateForm(): void {
    if (this.provider) {
      this.providerForm.patchValue({
        name: this.provider.name,
        endpoint: this.provider.endpoint,
        api_key_encrypted: this.provider.api_key_encrypted || '',
        description: this.provider.description || '',
        website: this.provider.website || '',
        support_email: this.provider.support_email || '',
        documentation_url: this.provider.documentation_url || '',
        is_active: this.provider.is_active ?? true,
        reliability_score: this.provider.reliability_score || 0,
        average_response_time_ms: this.provider.average_response_time_ms || 0
      });
    }
  }

  private resetForm(): void {
    this.providerForm.reset({
      name: '',
      endpoint: '',
      api_key_encrypted: '',
      description: '',
      website: '',
      support_email: '',
      documentation_url: '',
      is_active: true,
      reliability_score: 0,
      average_response_time_ms: 0
    });
  }

  onSubmit(): void {
    if (this.providerForm.invalid) {
      this.markFormGroupAsTouched(this.providerForm);
      return;
    }

    this.loading = true;
    this.error = null;

    const formValue = this.providerForm.value;
    const providerData: Partial<Provider> = {
      ...formValue,
      documentation_url: formValue.documentation_url || undefined
    };

    this.saved.emit(providerData as Provider);
  }

  onCancel(): void {
    this.cancelled.emit();
  }

  getErrorMessage(controlName: string): string {
    const control = this.providerForm.get(controlName);
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
    if (errors['email']) {
      return 'Please enter a valid email address';
    }
    if (errors['pattern']) {
      return 'Please enter a valid URL (must start with http:// or https://)';
    }

    return 'Invalid value';
  }

  private markFormGroupAsTouched(formGroup: FormGroup): void {
    Object.keys(formGroup.controls).forEach(key => {
      const control = formGroup.get(key);
      control?.markAsTouched();
    });
  }

  testConnection(): void {
    const endpoint = this.providerForm.get('endpoint')?.value;
    const apiKey = this.providerForm.get('api_key_encrypted')?.value;

    if (!endpoint || !apiKey) {
      this.error = 'Please provide both endpoint and API key to test connection';
      return;
    }

    this.loading = true;
    this.error = null;

    // In a real implementation, this would call a test endpoint
    // For now, we'll simulate a connection test
    setTimeout(() => {
      this.loading = false;
      // Simulate success (in real implementation, this would test the actual connection)
      console.log('Connection test would be performed here');
    }, 2000);
  }
}