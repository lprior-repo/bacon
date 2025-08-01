const { Tree, formatFiles, installPackagesTask, generateFiles, joinPathFragments } = require('@nx/devkit');

module.exports = async function(tree, options) {
  const { name, domain } = options;
  
  // Generate Go Lambda function
  generateFiles(
    tree,
    joinPathFragments(__dirname, 'files/lambda'),
    `src/${domain}/lambda/${name}`,
    {
      ...options,
      tmpl: ''
    }
  );
  
  // Generate Terraform module
  generateFiles(
    tree,
    joinPathFragments(__dirname, 'files/terraform'),
    `terraform/modules/lambda-${name}`,
    {
      ...options,
      tmpl: ''
    }
  );
  
  // Update main Terraform configuration
  const mainTf = tree.read('terraform/main.tf', 'utf-8');
  const newModule = `
module "lambda_${name}" {
  source = "./modules/lambda-${name}"
  
  namespace    = var.namespace
  environment  = var.environment
  
  depends_on = [
    module.dynamodb,
    module.vpc
  ]
}`;
  
  tree.write('terraform/main.tf', mainTf + newModule);
  
  console.log(`✅ Generated Lambda service: ${name}`);
  console.log(`   📁 Go code: src/${domain}/lambda/${name}/`);
  console.log(`   🏗️  Terraform: terraform/modules/lambda-${name}/`);
  console.log(`   🔗 Updated: terraform/main.tf`);
  
  await formatFiles(tree);
  return () => {
    console.log(`Run: nx build ${domain}-${name} to build the Lambda`);
    console.log(`Run: nx plan-dev infrastructure to plan Terraform changes`);
  };
};