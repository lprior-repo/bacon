const { execSync } = require('child_process');
const { join } = require('path');

module.exports = async function(options, context) {
  const { environment, workspace, autoApprove } = options;
  const projectRoot = context.workspace.projects[context.projectName].root;
  
  console.log(`🏗️  Planning Terraform for ${context.projectName} in ${environment}`);
  
  try {
    // Initialize if needed
    execSync('terraform init', { 
      cwd: projectRoot, 
      stdio: 'inherit',
      env: {
        ...process.env,
        TF_VAR_namespace: `${workspace}-${environment}`,
        TF_VAR_project_name: context.projectName
      }
    });
    
    // Run plan
    const planFile = `tfplan-${environment}-${Date.now()}`;
    execSync(`terraform plan -var-file=environments/${environment}.tfvars -out=${planFile}`, {
      cwd: projectRoot,
      stdio: 'inherit',
      env: {
        ...process.env,
        TF_VAR_namespace: `${workspace}-${environment}`,
        TF_VAR_project_name: context.projectName
      }
    });
    
    console.log(`✅ Plan complete: ${planFile}`);
    return { success: true, planFile };
    
  } catch (error) {
    console.error('❌ Terraform plan failed:', error.message);
    return { success: false, error: error.message };
  }
};