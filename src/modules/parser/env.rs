//! Environment variable substitution

use hyperterse_core::HyperterseError;
use once_cell::sync::Lazy;
use regex::Regex;

/// Regex pattern for environment variable placeholders: {{ env.VAR_NAME }}
static ENV_PATTERN: Lazy<Regex> = Lazy::new(|| {
    Regex::new(r"\{\{\s*env\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}").unwrap()
});

/// Environment variable substitutor
pub struct EnvSubstitutor {
    /// Whether to fail on missing environment variables
    strict: bool,
}

impl EnvSubstitutor {
    /// Create a new substitutor with strict mode (fails on missing vars)
    pub fn new() -> Self {
        Self { strict: true }
    }

    /// Create a new substitutor with lenient mode (leaves placeholders for missing vars)
    pub fn lenient() -> Self {
        Self { strict: false }
    }

    /// Substitute environment variables in the given content
    pub fn substitute(&self, content: &str) -> Result<String, HyperterseError> {
        // Load .env file if present (ignores errors)
        let _ = dotenvy::dotenv();

        let mut result = content.to_string();
        let mut errors: Vec<String> = Vec::new();

        // Find all matches and collect them first to avoid borrowing issues
        let matches: Vec<(String, String)> = ENV_PATTERN
            .captures_iter(content)
            .map(|cap| {
                let full_match = cap.get(0).unwrap().as_str().to_string();
                let var_name = cap.get(1).unwrap().as_str().to_string();
                (full_match, var_name)
            })
            .collect();

        for (full_match, var_name) in matches {
            match std::env::var(&var_name) {
                Ok(value) => {
                    result = result.replace(&full_match, &value);
                }
                Err(_) => {
                    if self.strict {
                        errors.push(var_name.clone());
                    }
                    // In lenient mode, leave the placeholder as-is
                }
            }
        }

        if !errors.is_empty() {
            return Err(HyperterseError::EnvVarNotFound(errors.join(", ")));
        }

        Ok(result)
    }

    /// Check if a string contains environment variable placeholders
    pub fn has_placeholders(content: &str) -> bool {
        ENV_PATTERN.is_match(content)
    }

    /// Extract all environment variable names from a string
    pub fn extract_var_names(content: &str) -> Vec<String> {
        ENV_PATTERN
            .captures_iter(content)
            .map(|cap| cap.get(1).unwrap().as_str().to_string())
            .collect()
    }
}

impl Default for EnvSubstitutor {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_has_placeholders() {
        assert!(EnvSubstitutor::has_placeholders("{{ env.DATABASE_URL }}"));
        assert!(EnvSubstitutor::has_placeholders("{{env.VAR}}"));
        assert!(EnvSubstitutor::has_placeholders("url: {{ env.DB_URL }}"));
        assert!(!EnvSubstitutor::has_placeholders("no placeholders"));
        assert!(!EnvSubstitutor::has_placeholders("{{ inputs.id }}"));
    }

    #[test]
    fn test_extract_var_names() {
        let content = "url: {{ env.DATABASE_URL }}, key: {{ env.API_KEY }}";
        let vars = EnvSubstitutor::extract_var_names(content);
        assert_eq!(vars.len(), 2);
        assert!(vars.contains(&"DATABASE_URL".to_string()));
        assert!(vars.contains(&"API_KEY".to_string()));
    }

    #[test]
    fn test_substitute_with_env_var() {
        std::env::set_var("TEST_VAR", "test_value");
        let substitutor = EnvSubstitutor::new();
        let result = substitutor.substitute("value: {{ env.TEST_VAR }}").unwrap();
        assert_eq!(result, "value: test_value");
        std::env::remove_var("TEST_VAR");
    }

    #[test]
    fn test_substitute_missing_var_strict() {
        let substitutor = EnvSubstitutor::new();
        let result = substitutor.substitute("{{ env.NONEXISTENT_VAR_12345 }}");
        assert!(result.is_err());
    }

    #[test]
    fn test_substitute_missing_var_lenient() {
        let substitutor = EnvSubstitutor::lenient();
        let result = substitutor
            .substitute("{{ env.NONEXISTENT_VAR_12345 }}")
            .unwrap();
        assert_eq!(result, "{{ env.NONEXISTENT_VAR_12345 }}");
    }
}
